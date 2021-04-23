// +build integration

package integration

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-jmx/tests/integration/helpers"
	"github.com/newrelic/nri-jmx/tests/integration/jsonschema"
	"github.com/stretchr/testify/assert"
)

var (
	defaultContainer = "integration_nri-jmx_1"
	defaultBinPath   = "/nri-jmx"

	jmx_host               = "tomcat"
	defaultCollectionFiles = "/jvm-metrics.yml,/tomcat-metrics.yml"

	// cli flags
	container = flag.String("container", defaultContainer, "container where the integration is installed")
	binPath   = flag.String("bin", defaultBinPath, "Integration binary path")

	collectionFiles = flag.String("collection_files", defaultCollectionFiles, "collection files")
)

// Returns the standard output, or fails testing if the command returned an error
func runIntegration(t *testing.T, envVars ...string) (string, string, error) {
	t.Helper()

	command := make([]string, 0)
	command = append(command, *binPath)

	var hasCollectionFiles bool

	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, "COLLECTION_FILES") {
			hasCollectionFiles = true
		}
	}

	if !hasCollectionFiles && collectionFiles != nil {
		command = append(command, "--collection_files", *collectionFiles)
	}
	command = append(command, "--jmx_host", jmx_host)

	stdout, stderr, err := helpers.ExecInContainer(*container, command, envVars...)

	if stderr != "" {
		log.Debug("Integration command Standard Error: ", stderr)
	}

	return stdout, stderr, err
}

func TestMain(m *testing.M) {
	flag.Parse()
	result := m.Run()
	os.Exit(result)
}

func TestJMXIntegration(t *testing.T) {
	stdout, stderr, err := runIntegration(t)

	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	schemaPath := filepath.Join("json-schema-files", "jmx-schema.json")
	err = jsonschema.Validate(schemaPath, stdout)
	assert.NoError(t, err, "The output of JMX integration doesn't have expected format.")
}

func TestJMXIntegration_ShowVersion(t *testing.T) {
	stdout, stderr, err := runIntegration(t, "SHOW_VERSION=true")
	assert.NotNil(t, stderr, "unexpected stderr")
	assert.NoError(t, err, "Unexpected error")

	expectedOutMessage := "New Relic Jmx integration Version: 0.0.0, Platform: linux/amd64, GoVersion: go1.16.3, GitCommit: , BuildDate: \n"
	assert.Equal(t, expectedOutMessage, stdout)
}

func TestJMXIntegration_ExceededMetricLimit(t *testing.T) {
	stdout, stderr, _ := runIntegration(t, "METRIC_LIMIT=1")

	expectedErrorMessage := "the current limit is 1. This Domain will not be reported"

	errMatch, _ := regexp.MatchString(expectedErrorMessage, stderr)
	assert.Truef(t, errMatch, "Expected error message: '%s', got: '%s'", expectedErrorMessage, stderr)

	assert.NotNil(t, stdout, "unexpected stdout")
}

func TestJMXIntegration_ErrorOpenFuncOnInvalidOptions(t *testing.T) {
	stdout, stderr, _ := runIntegration(t, "CONNECTION_URL=wrong_url")

	expectedErrorMessage := "Failed to complete collection: cannot query"

	errMatch, _ := regexp.MatchString(expectedErrorMessage, stderr)
	assert.Truef(t, errMatch, "Expected error message: '%s', got: '%s'", expectedErrorMessage, stderr)

	assert.NotNil(t, stdout, "unexpected stdout")
}

func TestJMXIntegration_ErrorEmptyCollectionFiles(t *testing.T) {
	stdout, stderr, err := runIntegration(t, "COLLECTION_FILES=")

	expectedErrorMessage := "Must specify at least one collection file"

	errMatch, _ := regexp.MatchString(expectedErrorMessage, stderr)
	assert.Error(t, err, "Expected error")
	assert.Truef(t, errMatch, "Expected error message: '%s', got: '%s'", expectedErrorMessage, stderr)

	assert.NotNil(t, stdout, "unexpected stdout")
}

func TestJMXIntegration_ErrorCollectionFileNotAbsolutePath(t *testing.T) {
	stdout, stderr, err := runIntegration(t, "COLLECTION_FILES=wrong_file.yml")

	expectedErrorMessage := "Metrics collection files must be specified as absolute paths"

	errMatch, _ := regexp.MatchString(expectedErrorMessage, stderr)
	assert.Error(t, err, "Expected error")
	assert.Truef(t, errMatch, "Expected error message: '%s', got: '%s'", expectedErrorMessage, stderr)

	assert.NotNil(t, stdout, "unexpected stdout")
}

func TestJMXIntegration_ErrorCollectionFileNotExisting(t *testing.T) {
	stdout, stderr, err := runIntegration(t, "COLLECTION_FILES=/wrong_file.yml")

	expectedErrorMessage := "Failed to parse collection definition"

	errMatch, _ := regexp.MatchString(expectedErrorMessage, stderr)
	assert.Error(t, err, "Expected error")
	assert.Truef(t, errMatch, "Expected error message: '%s', got: '%s'", expectedErrorMessage, stderr)

	assert.NotNil(t, stdout, "unexpected stdout")
}
