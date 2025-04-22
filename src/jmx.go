//go:generate goversioninfo
/*
 * Copyright 2022 New Relic Corporation. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/newrelic/nrjmx/gojmx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

const (
	integrationName = "com.newrelic.jmx"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	MetricLimit              int    `default:"200" help:"Number of metrics that can be collected per entity. If this limit is exceeded the entity will not be reported. A limit of 0 implies no limit."`
	Timeout                  int    `default:"10000" help:"Timeout for JMX queries"`
	JmxRemote                bool   `default:"false" help:"When activated uses the JMX remote url connection format (by default on JBoss Domain-mode)"`
	JmxRemoteJbossStandalone bool   `default:"false" help:"When activated uses the JMX remote url connection format on JBoss Standalone-mode"`
	JmxRemoteJbossStandlone  bool   `default:"false" help:"Deprecated, use -jmx-remote-jboss-standalone instead"`
	LocalEntity              bool   `default:"false" help:"Collect all metrics on the local entity. Use only when monitoring localhost."`
	RemoteMonitoring         bool   `default:"false" help:"Allows to monitor multiple instances as 'remote' entity. Set to 'FALSE' value for backwards compatibility otherwise set to 'TRUE'"`
	JmxSSL                   bool   `default:"false" help:"Use https"`
	ShowVersion              bool   `default:"false" help:"Print build information and exit"`
	HideSecrets              bool   `default:"true" help:"Set this to false if you want to see the secrets in the verbose logs."`
	KeyStore                 string `default:"" help:"The location for the keystore containing JMX Client's SSL certificate"`
	KeyStorePassword         string `default:"" help:"Password for the SSL Key Store"`
	TrustStore               string `default:"" help:"The location for the keystore containing JMX Server's SSL certificate"`
	TrustStorePassword       string `default:"" help:"Password for the SSL Trust Store"`
	CollectionFiles          string `default:"" help:"A comma separated list of full paths to metrics collections configuration files"`
	CollectionConfig         string `default:"" help:"JSON format metrics collection configuration"`
	NrJmx                    string `default:"/usr/bin/nrjmx" help:"nrjmx tool executable path"`
	ConnectionURL            string `default:"" help:"full connection URL"`
	Query                    string `default:"" help:"For troubleshooting only: Connect to the JMX endpoint and execute the query. Query format DOMAIN:BEAN"`
	ConfigFile               string `default:"/etc/newrelic-infra/integrations.d/jmx-config.yml" help:"For troubleshooting only: Specify JMX config file. If you don't want to load the config from the file set this empty"`
	InstanceName             string `default:"" help:"For troubleshooting only: Specify which block from the jmx config file will be used. You can find the value in the jmx config file. Is the name field of the instance / integration. If left empty, first configuration block will be used."`
	JmxHost                  string `default:"localhost" help:"The host running JMX"`
	JmxPort                  string `default:"9999" help:"The port JMX is running on"`
	JmxURIPath               string `default:"" help:"The path portion of the JMX Service URI. This is useful for nonstandard service uris"`
	JmxUser                  string `default:"" help:"The username for the JMX connection"`
	JmxPass                  string `default:"" help:"The password for the JMX connection"`
	LongRunning              bool   `default:"false" help:"BETA: In long-running mode integration process will be kept alive"`
	HeartbeatInterval        int    `default:"5" help:"BETA: Interval in seconds for submitting the heartbeat while in long-running mode"`
	Interval                 int    `default:"30" help:"BETA: Interval in seconds for collecting data while while in long-running mode"`
	EnableInternalStats      bool   `default:"false" help:"Print nrjmx internal query stats for troubleshooting"`
}

var (
	args               argumentList
	integrationVersion = "0.0.0"
	gitCommit          = ""
	buildDate          = ""

	errNRJMXNotRunning = errors.New("nrjmx client sub-process not running")
)

func main() {
	// Create a new integration
	jmxIntegration, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	fatalIfErr(err)

	jmxClient := gojmx.NewClient(context.Background())

	// Troubleshooting mode, we need to read the args from the configuration file.
	if args.Query != "" {
		err = SetArgs(args.InstanceName, args.ConfigFile)
		fatalIfErr(err)

		result := FormatQuery(jmxClient, getJMXConfig(), args.Query, args.HideSecrets)
		fmt.Println(result)
		os.Exit(0)
	}

	if args.ShowVersion {
		caser := cases.Title(language.English)
		fmt.Printf(
			"New Relic %s integration Version: %s, Platform: %s, GoVersion: %s, GitCommit: %s, BuildDate: %s\n",
			caser.String(strings.Replace(integrationName, "com.newrelic.", "", 1)),
			integrationVersion,
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			runtime.Version(),
			gitCommit,
			buildDate)
		os.Exit(0)
	}

	log.SetupLogging(args.Verbose)

	// Ensure a collection file is specified
	if args.CollectionFiles == "" && args.CollectionConfig == "" {
		log.Error("Must specify at least one collection file or a collection config JSON")
		os.Exit(1)
	}

	jmxClient, err = openJMXConnection()
	if err != nil {
		log.Error("Failed to open JMX connection, error: %v, Config: (%s)",
			err,
			gojmx.FormatConfig(getJMXConfig(), args.HideSecrets),
		)
		os.Exit(1)
	}

	err = runMetricCollection(jmxIntegration, jmxClient)

	// Make sure we close the connection after collection was done.
	// We cannot defer this, since we are using log.Fail/os.Exit
	if connErr := jmxClient.Close(); connErr != nil {
		log.Error(
			"Failed to close JMX connection: %s", err)
	}

	fatalIfErr(err)

	jmxIntegration.Entities = checkMetricLimit(jmxIntegration.Entities)

	if err := jmxIntegration.Publish(); err != nil {
		log.Error("Failed to publish integration: %s", err.Error())
		os.Exit(1)
	}
}

// runCollectionFiles will run the collection for collection files configuration.
func runCollectionFiles(jmxIntegration *integration.Integration, client Client) {
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

		if err := runCollection(collection, jmxIntegration, client, args.JmxHost, args.JmxPort); err != nil {
			log.Error("Failed to complete collection: %s", err)
		}
	}
}

// runCollectionConfig will run the collection for JSON collection configuration
func runCollectionConfig(jmxIntegration *integration.Integration, client Client) {
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

	if err := runCollection(collection, jmxIntegration, client, args.JmxHost, args.JmxPort); err != nil {
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

// runMetricCollection will perform the metrics collection.
func runMetricCollection(i *integration.Integration, jmxClient *gojmx.Client) error {
	if args.LongRunning {
		return collectMetricsEachInterval(i, jmxClient)
	}
	return collectMetrics(i, jmxClient)
}

// collectMetricsEachInterval will collect the metrics periodically when configured in long-running mode.
func collectMetricsEachInterval(i *integration.Integration, jmxClient *gojmx.Client) error {
	metricInterval := time.NewTicker(time.Duration(args.Interval) * time.Second)

	runHeartBeat()

	// do ... while.
	for ; true; <-metricInterval.C {
		// Check if the nrjmx java sub-process is still alive.
		if !jmxClient.IsRunning() {
			return errNRJMXNotRunning
		}

		if err := collectMetrics(i, jmxClient); err != nil {
			log.Error("Failed to collect metrics, error: %v", err)
			continue
		}

		if err := i.Publish(); err != nil {
			log.Error("Failed to publish metrics, error: %v", err)
			continue
		}
	}

	return nil
}

// collectMetrics will gather all the required metrics from the JMX endpoint and attach them the the sdk integration.
func collectMetrics(i *integration.Integration, jmxClient *gojmx.Client) error {
	// For troubleshooting purpose, if enabled, integration will log internal query stats.
	if args.EnableInternalStats {
		defer func() {
			logInternalStats(jmxClient)
		}()
	}

	runCollectionFiles(i, jmxClient)
	runCollectionConfig(i, jmxClient)

	return nil
}

// runHeartBeat is used in long-running mode to signal to the agent that the integration is alive.
func runHeartBeat() {
	heartBeat := time.NewTicker(time.Duration(args.HeartbeatInterval) * time.Second)

	go func() {
		for range heartBeat.C {
			log.Debug("Sending heartBeat")
			// heartbeat signal for long-running integrations
			// https://docs.newrelic.com/docs/integrations/integrations-sdk/file-specifications/host-integrations-newer-configuration-format#timeout
			fmt.Println("{}")
		}
	}()
}

// logInternalStats will print in verbose logs statistics gathered by nrjmx client
// that can be handy when troubleshooting performance issues.
func logInternalStats(jmxClient *gojmx.Client) {
	internalStats, err := jmxClient.GetInternalStats()
	if err != nil {
		log.Error("Failed to collect nrjmx internal stats, %v", err)
		return
	}

	for _, stat := range internalStats {
		log.Debug("%v", stat)
	}

	// Aggregated stats.
	log.Debug("%v", internalStats)
}

// openJMXConnection configures the JMX client and attempts to connect to the endpoint.
func openJMXConnection() (*gojmx.Client, error) {
	jmxConfig := getJMXConfig()

	hideSecrets := true
	formattedConfig := gojmx.FormatConfig(jmxConfig, hideSecrets)

	jmxClient := gojmx.NewClient(context.Background())
	_, err := jmxClient.Open(jmxConfig)

	log.Debug("nrjmx version: %s, config: %s", jmxClient.GetClientVersion(), formattedConfig)

	if err != nil {
		// When not in long-running mode, we cannot recover from any type of connection error.
		// However, in long-running mode, we can recover later from errors related with connection, except JMXClient error
		// which means that the nrjmx java sub-process was closed.
		if _, ok := gojmx.IsJMXClientError(err); ok || !args.LongRunning {
			return nil, fmt.Errorf("failed to open JMX connection, error: %w, Config: (%s)",
				err,
				formattedConfig,
			)
		}

		// In long-running mode just log the error.
		log.Error("Error while connecting to jmx connection, err: %v", err)
	}

	return jmxClient, nil
}

func getJMXConfig() *gojmx.JMXConfig {
	port, err := strconv.ParseInt(args.JmxPort, 10, 32) //nolint
	if err != nil {
		log.Error("Failed to parse JMX port argument: %v", err)
	}
	jmxConfig := &gojmx.JMXConfig{
		ConnectionURL:         args.ConnectionURL,
		IsRemote:              args.JmxRemote,
		IsJBossStandaloneMode: args.JmxRemoteJbossStandlone || args.JmxRemoteJbossStandalone,
		KeyStore:              args.KeyStore,
		KeyStorePassword:      args.KeyStorePassword,
		TrustStore:            args.TrustStore,
		TrustStorePassword:    args.TrustStorePassword,
		Hostname:              args.JmxHost,
		Port:                  int32(port),
		Username:              args.JmxUser,
		Password:              args.JmxPass,
		RequestTimeoutMs:      int64(args.Timeout),
		UseSSL:                args.JmxSSL,
		Verbose:               args.Verbose,
		EnableInternalStats:   args.EnableInternalStats,
	}
	if args.JmxURIPath != "" {
		jmxConfig.UriPath = &(args.JmxURIPath)
	}
	return jmxConfig
}

func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
