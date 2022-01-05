package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nrjmx/gojmx"
	yaml "gopkg.in/yaml.v3"
)

var (
	ErrConfig = errors.New("config error")
)

// FormatQuery is used for troubleshooting. It formats the result in multiple yaml sections.
func FormatQuery(client Client, config *gojmx.JMXConfig, mBeanGlobPattern string, hideSecrets bool) string {
	sb := strings.Builder{}
	sb.WriteString("=======================================================\n")
	sb.WriteString("Connecting to JMX...\n\n")
	sb.WriteString("Config: " + gojmx.FormatConfig(config, hideSecrets) + "\n\n")
	if _, err := client.Open(config); err != nil {
		sb.WriteString("Error: " + err.Error() + "\n")
		return sb.String()
	}

	defer func() {
		if err := client.Close(); err != nil {
			log.Error(
				"Failed to close JMX connection: %s", err)
		}
	}()

	sb.WriteString("Connected!\n")

	response, err := client.QueryMBean(mBeanGlobPattern)
	if err != nil {
		sb.WriteString("Error: " + err.Error() + "\n")
		return sb.String()
	}
	sb.WriteString(gojmx.FormatJMXAttributes(response.GetValidAttributes()))

	return sb.String()
}

// SetArgs will read the config file and will set the integration flags. This is used for troubleshooting.
func SetArgs(integrationName, configFile string) error {
	if configFile == "" {
		return nil
	}
	result, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	return parseArgsFromConfig(result, integrationName, configFile)
}

func parseArgsFromConfig(config []byte, integrationName, fileName string) error {
	cfg := configFile{
		integrationName: integrationName,
		fileName:        fileName,
	}

	if err := yaml.Unmarshal(config, &cfg); err != nil {
		return err
	}

	configOptions, err := cfg.toConfigOptions()
	if err != nil {
		return err
	}

	return setArgs(configOptions)
}

func setArgs(configOptions map[string]interface{}) error {
	for optionName, option := range configOptions {
		os.Setenv(strings.ToUpper(optionName), fmt.Sprintf("%v", option))
	}

	// Overwrite all flag values with env vars.
	flag.VisitAll(func(f *flag.Flag) {
		envName := strings.ToUpper(f.Name)
		if os.Getenv(envName) != "" {
			f.Value.Set(os.Getenv(envName)) // nolint: errcheck
		}
	})
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return err
	}
	return nil
}

type instance struct {
	Name      string
	Arguments map[string]interface{}
}

// For integrations v4.
type integrations struct {
	Name string
	Env  map[string]interface{}
}

type configFile struct {
	fileName        string
	integrationName string
	Instances       []instance
	Integrations    []integrations
}

func (c *configFile) toConfigOptions() (map[string]interface{}, error) {
	hasInstances := len(c.Instances) > 0
	hasIntegrations := len(c.Integrations) > 0

	if !hasInstances && !hasIntegrations {
		return nil, fmt.Errorf("%w: failed to detect any integration in the config file: '%s'", ErrConfig, c.fileName)
	}

	if hasInstances {
		return c.getInstances()
	}
	return c.getIntegrations()
}

func (c *configFile) getInstances() (map[string]interface{}, error) {
	if c.integrationName == "" {
		return c.Instances[0].Arguments, nil
	}
	var configOptions map[string]interface{}

	for _, instance := range c.Instances {
		if instance.Name == c.integrationName {
			configOptions = instance.Arguments
			break
		}
	}
	if configOptions == nil {
		return nil, fmt.Errorf("%w: failed to detect instance: '%s' in file: '%s'", ErrConfig, c.integrationName, c.fileName)
	}

	return configOptions, nil
}

func (c *configFile) getIntegrations() (map[string]interface{}, error) {
	if c.integrationName == "" {
		return c.Integrations[0].Env, nil
	}
	var configOptions map[string]interface{}

	for _, integration := range c.Integrations {
		if integration.Name == c.integrationName {
			configOptions = integration.Env
		}
	}
	if configOptions == nil {
		return nil, fmt.Errorf("%w: failed to detect integration: '%s' in file: '%s'", ErrConfig, c.integrationName, c.fileName)
	}
	return configOptions, nil
}
