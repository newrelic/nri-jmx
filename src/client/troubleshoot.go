package client

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/newrelic/nrjmx/gojmx"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"reflect"
	"strings"
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
	defer jmxClient.Disconnect()
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
	result, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	return parseConfigToArgs(args, result, integrationName, configFile)
}

func parseConfigToArgs(args interface{}, config []byte, integrationName, fileName string) error {
	t := struct {
		Instances []struct {
			Name      string
			Arguments map[string]interface{}
		}
		// For integrations v4.
		Integrations []struct {
			Name string
			Env  map[string]interface{}
		}
	}{}

	err := yaml.Unmarshal(config, &t)
	if err != nil {
		return err
	}

	hasInstances := len(t.Instances) > 0
	hasIntegrations := len(t.Integrations) > 0

	if !hasInstances && !hasIntegrations {
		return fmt.Errorf("failed to detect any integration in the config file: '%s'", fileName)
	}

	var configOptions map[string]interface{}
	if hasInstances {
		if integrationName == "" {
			configOptions = t.Instances[0].Arguments
		} else {
			for _, instance := range t.Instances {
				if instance.Name == integrationName {
					configOptions = instance.Arguments
					break
				}
			}
			if configOptions == nil {
				return fmt.Errorf("failed to detect instance: '%s' in file: '%s'", integrationName, fileName)
			}
		}
	} else if hasIntegrations {
		if integrationName == "" {
			configOptions = t.Integrations[0].Env
		} else {
			for _, integration := range t.Integrations {
				if integration.Name == integrationName {
					configOptions = integration.Env
				}
			}
			if configOptions == nil {
				return fmt.Errorf("failed to detect integration: '%s' in file: '%s'", integrationName, fileName)
			}
		}
	}

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
