package main

import (
	"fmt"
	"os"
	"strings"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	JmxHost         string `default:"localhost" help:"The host running JMX"`
	JmxPort         string `default:"9999" help:"The port JMX is running on"`
	JmxUser         string `default:"admin" help:"The username for the JMX connection"`
	JmxPass         string `default:"admin" help:"The password for the JMX connection"`
	CollectionFiles string `default:"" help:"A comma separated list of full paths to metrics configuration files"`
	Timeout         int    `default:"10000" help:"Timeout for JMX queries"`
}

const (
	integrationName    = "com.newrelic.jmx"
	integrationVersion = "0.1.0"
)

var (
	args argumentList

	jmxOpenFunc  = jmx.Open
	jmxCloseFunc = jmx.Close
	jmxQueryFunc = jmx.Query
)

func main() {

	// Create a new integration
	jmxIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	if err != nil {
		os.Exit(1)
	}

	// Open a JMX connection
	if err := jmxOpenFunc(args.JmxHost, args.JmxPort, args.JmxUser, args.JmxPass); err != nil {
		log.Error(
			"Failed to open JMX connection (host: %s, port: %s, user: %s, pass: %s): %s",
			args.JmxHost, args.JmxPort, args.JmxUser, args.JmxPass, err,
		)
		os.Exit(1)
	}

	if args.CollectionFiles == "" {
		log.Error("Must specify at least one collection file")
		os.Exit(1)
	}

	// For each collection definition file, parse and collect it
	collectionFiles := strings.Split(args.CollectionFiles, ",")
	for _, collectionFile := range collectionFiles {

		// Parse the yaml file into a raw definition
		collectionDefinition, err := parseYaml(collectionFile)
		if err != nil {
			log.Error("Failed to parse collection definition file %s: %s", collectionFile, err)
			os.Exit(1)
		}

		// Validate the definition and create a collection object
		collection, err := parseCollectionDefinition(collectionDefinition)
		if err != nil {
			log.Error("Failed to parse collection definition %s: %s", collectionFile, err)
			os.Exit(1)
		}

		if err := runCollection(collection, jmxIntegration); err != nil {
			log.Error("Failed to complete collection: %s", err)
		}
	}

	jmxCloseFunc()

	if err := jmxIntegration.Publish(); err != nil {
		os.Exit(1)
	}
}
