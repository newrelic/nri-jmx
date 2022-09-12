package helpers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-jmx/test/integration/jsonschema"
	"github.com/stretchr/testify/assert"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ExecInContainer executes the given command inside the specified container. It returns three values:
// 1st - Standard Output
// 2nd - Standard Error
// 3rd - Runtime error, if any
func ExecInContainer(container string, command []string, envVars ...string) (string, string, error) {
	cmdLine := make([]string, 0, 3+len(command))
	cmdLine = append(cmdLine, "exec", "-i")

	for _, envVar := range envVars {
		cmdLine = append(cmdLine, "-e", envVar)
	}

	cmdLine = append(cmdLine, container)
	cmdLine = append(cmdLine, command...)

	log.Debug("executing: docker %s", strings.Join(cmdLine, " "))

	cmd := exec.Command("docker", cmdLine...)

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout := outbuf.String()
	stderr := errbuf.String()

	return stdout, stderr, err
}

// NewDockerExecCommand returns a configured un-started exec.Cmd for a docker exec command.
func NewDockerExecCommand(ctx context.Context, t *testing.T, containerName string, args []string, envVars map[string]string) *exec.Cmd {
	cmdLine := []string{
		"exec",
		"-i",
	}

	for key, val := range envVars {
		cmdLine = append(cmdLine, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	cmdLine = append(cmdLine, containerName)
	cmdLine = append(cmdLine, args...)

	t.Logf("executing: docker %s", strings.Join(cmdLine, " "))

	return exec.CommandContext(ctx, "docker", cmdLine...)
}

// Output for a long-running docker exec command.
type Output struct {
	StdoutCh chan string
	StderrCh chan string
}

// NewOutput returns a new Output object.
func NewOutput() *Output {
	size := 1000
	return &Output{
		StdoutCh: make(chan string, size),
		StderrCh: make(chan string, size),
	}
}

// Flush will empty the Output channels and returns the content.
func (o *Output) Flush(t *testing.T) (stdout []string, stderr []string) {
	for {
		select {
		case line := <-o.StdoutCh:
			t.Logf("Flushing stdout line: %s", line)
			stdout = append(stdout, line)
		case line := <-o.StderrCh:
			t.Logf("Flushing stderr line: %s", line)
			stderr = append(stderr, line)
		default:
			return
		}
	}
}

// StartLongRunningProcess will execute a command and will pipe the stdout & stderr into and Output object.
func StartLongRunningProcess(ctx context.Context, t *testing.T, cmd *exec.Cmd) (*Output, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	copyToChan := func(ctx context.Context, reader io.Reader, outputC chan string) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() && ctx.Err() == nil {
			outputC <- scanner.Text()
		}

		if err := scanner.Err(); ctx.Err() == nil && err != nil {
			t.Logf("Error while reading the pipe, %v", err)
			return
		}

		t.Log("Finished reading the pipe")
	}

	output := NewOutput()

	go copyToChan(ctx, stdout, output.StdoutCh)
	go copyToChan(ctx, stderr, output.StderrCh)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return output, nil
}

// RunDockerCommandForContainer will execute a docker command for the specified containerName.
func RunDockerCommandForContainer(t *testing.T, command, containerName string) error {
	t.Logf("running docker %s container %s", command, containerName)

	cmd := exec.Command("docker", command, containerName)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("errror while %s the container '%s', error: %v, stderr: %s", command, containerName, err, errBuf.String())
	}

	return nil
}

// AssertReceivedErrors check if at least one the log lines provided contains the given message.
func AssertReceivedErrors(t *testing.T, msg string, errLog ...string) {
	assert.GreaterOrEqual(t, len(errLog), 1)

	for _, line := range errLog {
		if strings.Contains(line, msg) {
			return
		}
	}

	assert.Failf(t, fmt.Sprintf("Expected to find the following error message: %s", msg), "but got %s", errLog)
}

// AssertReceivedPayloadsMatchSchema will check if payloads inside Output object matches the give JSON schema.
func AssertReceivedPayloadsMatchSchema(t *testing.T, ctx context.Context, output *Output, schemaPath string, timeout time.Duration) {
	var cancelFn context.CancelFunc

	ctx, cancelFn = context.WithTimeout(ctx, timeout)
	defer cancelFn()

	validPayloads := 0
	validHeartbeats := 0

	for {
		if validPayloads >= 3 && validHeartbeats >= 3 {
			break
		}

		select {
		case stdoutLine := <-output.StdoutCh:
			if stdoutLine == "{}" {
				t.Log("Received heartbeat")
				validHeartbeats++
			} else {
				t.Logf("Received payload: %s", stdoutLine)

				err := jsonschema.Validate(schemaPath, stdoutLine)
				if err == nil {
					validPayloads++
				}
				assert.NoError(t, err)
			}

		case stderrLine := <-output.StderrCh:
			t.Logf("Received stderr: %s", stderrLine)

			assert.Empty(t, FilterStderr(stderrLine))
		case <-ctx.Done():
			assert.FailNow(t, "didn't received output in time")
		}
	}
}

// FilterStderr is handy to filter some log lines that are expected.
func FilterStderr(content string) string {
	return FilterLines(content, ExpectedErrMessagesFilter)
}

func FilterLines(content string, filter func(line string) bool) string {
	if content == "" {
		return content
	}
	var result []string
	for _, line := range strings.Split(content, "\n") {
		if !filter(line) {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func ExpectedErrMessagesFilter(line string) bool {
	wordsToIgnoreLines := []string{
		"[INFO]",
		"[DEBUG]",
		"non-numeric value for gauge metric",
	}
	for _, chunk := range wordsToIgnoreLines {
		if strings.Contains(line, chunk) {
			return true
		}
	}
	return false
}
