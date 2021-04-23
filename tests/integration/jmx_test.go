// +build integration

package integration

import (
	"flag"
	"os"
	"path/filepath"
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
