//go:generate goversioninfo
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/jmx"
	"github.com/newrelic/infra-integrations-sdk/log"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	JmxHost                 string `default:"localhost" help:"The host running JMX"`
	JmxPort                 string `default:"9999" help:"The port JMX is running on"`
	JmxURIPath              string `default:"" help:"The path portion of the JMX Service URI. This is useful for nonstandard service uris"`
	JmxUser                 string `default:"" help:"The username for the JMX connection"`
	JmxPass                 string `default:"" help:"The password for the JMX connection"`
	JmxRemote               bool   `default:"false" help:"When activated uses the JMX remote url connection format (by default on JBoss Domain-mode)"`
	JmxRemoteJbossStandlone bool   `default:"false" help:"When activated uses the JMX remote url connection format on JBoss Standalone-mode"`
	LocalEntity             bool   `default:"false" help:"Collect all metrics on the local entity. Use only when monitoring localhost."`
	RemoteMonitoring        bool   `default:"false" help:"Allows to monitor multiple instances as 'remote' entity. Set to 'FALSE' value for backwards compatibility otherwise set to 'TRUE'"`
	KeyStore                string `default:"" help:"The location for the keystore containing JMX Client's SSL certificate"`
	KeyStorePassword        string `default:"" help:"Password for the SSL Key Store"`
	TrustStore              string `default:"" help:"The location for the keystore containing JMX Server's SSL certificate"`
	TrustStorePassword      string `default:"" help:"Password for the SSL Trust Store"`
	CollectionFiles         string `default:"" help:"A comma separated list of full paths to metrics collections configuration files"`
	CollectionConfig        string `default:"" help:"JSON format metrics collection configuration"`
	Timeout                 int    `default:"10000" help:"Timeout for JMX queries"`
	MetricLimit             int    `default:"200" help:"Number of metrics that can be collected per entity. If this limit is exceeded the entity will not be reported. A limit of 0 implies no limit."`
	NrJmx                   string `default:"/usr/bin/nrjmx" help:"nrjmx tool executable path"`
	ConnectionURL           string `default:"" help:"full connection URL"`
	ShowVersion             bool   `default:"false" help:"Print build information and exit"`
}

const (
	integrationName = "com.newrelic.jmx"
)

var (
	args argumentList

	jmxOpenFunc  = jmx.Open
	jmxCloseFunc = jmx.Close
	jmxQueryFunc = jmx.Query

	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""
)

func main() {

	// Create a new integration
	jmxIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	if err != nil {
		os.Exit(1)
	}

	if args.ShowVersion {
		fmt.Printf(
			"New Relic %s integration Version: %s, Platform: %s, GoVersion: %s, GitCommit: %s, BuildDate: %s\n",
			strings.Title(strings.Replace(integrationName, "com.newrelic.", "", 1)),
			integrationVersion,
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			runtime.Version(),
			gitCommit,
			buildDate)
		os.Exit(0)
	}

	log.SetupLogging(args.Verbose)

	options := []jmx.Option{
		jmx.WithNrJmxTool(args.NrJmx),
	}

	if args.Verbose {
		options = append(options, jmx.WithVerbose())
	}

	if args.ConnectionURL != "" {
		options = append(options, jmx.WithConnectionURL(args.ConnectionURL))
	} else {
		if args.JmxURIPath != "" {
			options = append(options, jmx.WithURIPath(args.JmxURIPath))
		}
		if args.JmxRemote {
			options = append(options, jmx.WithRemoteProtocol())
			if args.JmxRemoteJbossStandlone {
				options = append(options, jmx.WithRemoteStandAloneJBoss())
			}
		}
	}

	if args.KeyStore != "" && args.KeyStorePassword != "" && args.TrustStore != "" && args.TrustStorePassword != "" {
		ssl := jmx.WithSSL(args.KeyStore, args.KeyStorePassword, args.TrustStore, args.TrustStorePassword)
		options = append(options, ssl)
	}
	if err := jmxOpenFunc(args.JmxHost, args.JmxPort, args.JmxUser, args.JmxPass, options...); err != nil {
		log.Error(
			"Failed to open JMX connection (host: %s, port: %s, user: %s, pass: %s, keyStore: %s, keyStorePassword: %s, trustStore: %s, trustStorePassword: %s, remote: %t): %s",
			args.JmxHost, args.JmxPort, args.JmxUser, args.JmxPass, args.KeyStore, args.KeyStorePassword, args.TrustStore, args.TrustStorePassword, args.JmxRemote, err,
		)
		os.Exit(1)
	}

	// Ensure a collection file is specified
	if args.CollectionFiles == "" && args.CollectionConfig == "" {
		log.Error("Must specify at least one collection file or a collection config JSON")
		os.Exit(1)
	}

	runCollectionFiles(jmxIntegration)
	runCollectionConfig(jmxIntegration)

	jmxCloseFunc()

	jmxIntegration.Entities = checkMetricLimit(jmxIntegration.Entities)

	if err := jmxIntegration.Publish(); err != nil {
		log.Error("Failed to publish integration: %s", err.Error())
		os.Exit(1)
	}
}

// runCollectionFiles will run the collection for collection files configuration.
func runCollectionFiles(jmxIntegration *integration.Integration) {
	if args.CollectionFiles == "" {
		return
	}
	// For each collection definition file, parse and collect it
	collectionFiles := strings.Split(args.CollectionFiles, ",")

	for _, collectionFile := range collectionFiles {
		// Check that the filepath is an absolute path
		if !filepath.IsAbs(collectionFile) {
			log.Error("Invalid metrics collection path %s. Metrics collection files must be specified as absolute paths.", collectionFile)
			os.Exit(1)
		}

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

		if err := runCollection(collection, jmxIntegration, args.JmxHost, args.JmxPort); err != nil {
			log.Error("Failed to complete collection: %s", err)
		}
	}
}

// runCollectionConfig will run the collection for JSON collection configuration
func runCollectionConfig(jmxIntegration *integration.Integration) {
	if args.CollectionConfig == "" {
		return
	}

	// Parse the JSON collection config into a raw definition
	collectionDefinition, err := parseJSON(args.CollectionConfig)
	if err != nil {
		log.Error("Failed to parse collection definition config %s: %s", args.CollectionConfig, err)
		os.Exit(1)
	}

	// Validate the definition and create a collection object
	collection, err := parseCollectionDefinition(collectionDefinition)
	if err != nil {
		log.Error("Failed to parse collection definition config %s: %s", args.CollectionConfig, err)
		os.Exit(1)
	}

	if err := runCollection(collection, jmxIntegration, args.JmxHost, args.JmxPort); err != nil {
		log.Error("Failed to complete collection: %s", err)
	}
}

// checkMetricLimit looks through all of the metric sets for every entity and aggregates the number
// of metrics. If that total is greate than args.MetricLimit a warning is logged
func checkMetricLimit(entities []*integration.Entity) []*integration.Entity {
	validEntities := make([]*integration.Entity, 0, len(entities))

	for _, entity := range entities {
		metricCount := 0
		for _, metricSet := range entity.Metrics {
			metricCount += len(metricSet.Metrics)
		}

		if args.MetricLimit != 0 && metricCount > args.MetricLimit {
			log.Warn("Domain '%s' has %d metrics, the current limit is %d. This Domain will not be reported", entity.Metadata.Name, metricCount, args.MetricLimit)
			continue
		}

		validEntities = append(validEntities, entity)
	}

	return validEntities
}
