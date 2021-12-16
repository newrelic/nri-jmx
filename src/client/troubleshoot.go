package client

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nrjmx/gojmx"
	"gopkg.in/yaml.v3"
)

func FormatQuery(mBeanGlobPattern string, config *gojmx.JMXConfig, hideSecrets bool) string {
	sb := strings.Builder{}
	sb.WriteString("=======================================================\n")
	sb.WriteString("Connecting to JMX...\n\n")
	sb.WriteString("Config: " + gojmx.FormatConfig(config, hideSecrets) + "\n\n")
	jmxClient := NewJMXClient()
	err := jmxClient.Connect(config)
	if err != nil {
		sb.WriteString("Error: " + err.Error() + "\n")
	}

	go func() {
		if err := jmxClient.Disconnect(); err != nil {
			log.Error(
				"Failed to close JMX connection: %s", err)
		}
	}()

	sb.WriteString("Connected!\n")

	attrs, err := jmxClient.QueryMBean(mBeanGlobPattern)
	if err != nil {
		sb.WriteString("Error: " + err.Error() + "\n")
	}

	sb.WriteString(gojmx.FormatJMXAttributes(attrs))

	return sb.String()
}

func SetArgs(args interface{}, integrationName, configFile string) error {
	if configFile == "" {
		return nil
	}
	result, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	return parseConfigToArgs(args, result, integrationName, configFile)
}

func parseConfigToArgs(args interface{}, config []byte, integrationName, fileName string) error {
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

	return setArgs(args, fileName, configOptions)
}

func setArgs(args interface{}, fileName string, configOptions map[string]interface{}) error {
	r := reflect.ValueOf(args)
	for optionName, option := range configOptions {
		camelCase := strcase.ToCamel(strings.ToLower(optionName))
		fieldByName := reflect.Indirect(r).FieldByName(camelCase)
		if !fieldByName.IsValid() {
			return fmt.Errorf("failed to parse config field: '%s' from file: '%s'", optionName, fileName)
		}

		fieldByName.Set(reflect.ValueOf(option))
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
		return nil, fmt.Errorf("failed to detect any integration in the config file: '%s'", c.fileName)
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
		return nil, fmt.Errorf("failed to detect instance: '%s' in file: '%s'", c.integrationName, c.fileName)
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
		return nil, fmt.Errorf("failed to detect integration: '%s' in file: '%s'", c.integrationName, c.fileName)
	}
	return configOptions, nil
}
