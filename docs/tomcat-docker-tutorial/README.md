# Collecting JMX metrics from a Tomcat v9.0.8 running inside docker container

## Prerequisites

## 1. <a name='InstalltheInfrastructureagent'></a>Install the Infrastructure agent and JMX integration

- [Install Infrastructure for Linux using the package manager](https://docs.newrelic.com/docs/infrastructure/install-configure-manage-infrastructure/linux-installation/install-infrastructure-linux-using-package-manager)

  or

- [Install Infrastructure for Windows Server using the MSI installer](https://docs.newrelic.com/docs/infrastructure/install-configure-manage-infrastructure/windows-installation/install-infrastructure-windows-server-using-msi-installer)

- [Install New Relic JMX integration](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/jmx-monitoring-integration#install)

## 2. Expose JMX from Tomcat

For this tutorial we will run a Tomcat in inside Docker using the following Dockerfile:

```bash

FROM tomcat:9.0.8

ARG CATALINA_OPTS
ENV CATALINA_OPTS="-Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.local.only=false -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=9010 -Dcom.sun.management.jmxremote.rmi.port=9010 -Djava.rmi.server.hostname=0.0.0.0 -Dcom.sun.management.jmxremote.ssl=false"

EXPOSE 9010
EXPOSE 8080
```

Build and run the image, exposing the JMX port 9010:

```bash
docker build -t tomcat_908_jmx . && docker run -d -p 9010:9010 --name=tomcat_908_jmx tomcat_908_jmx
```

## 3. Configure JMX integration

### 3.1 First step is creating a JMX integration configuration file `/etc/newrelic-infra/integrations.d/jmx-config.yml`

for the jmx_host use this command to obtain ip address of the container `docker inspect --format '{{ .NetworkSettings.IPAddress }}' tomcat_908_jmx` (binding to 0.0.0.0 not working in this case)

```yaml
integrations:
  - name: nri-jmx
    env:
      COLLECTION_FILES: "/etc/newrelic-infra/integrations.d/tomcat-metrics.yml"
      JMX_HOST: localhost
      JMX_PORT: "9010"

      # New users should leave this property as `true`, to identify the
      # monitored entities as `remote`. Setting this property to `false` (the
      # default value) is deprecated and will be removed soon, disallowing
      # entities that are identified as `local`.
      # Please check the documentation to get more information about local
      # versus remote entities:
      # https://github.com/newrelic/infra-integrations-sdk/blob/master/docs/entity-definition.md
      REMOTE_MONITORING: "true"
    interval: 15s
    labels:
      env: staging
```

All configuration options can be found in the public [documentation](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/jmx-monitoring-integration#config).

### Test the JMX connection

`nri-jmx` `query` tool uses the defined jmx-config.yml file to establish connection and  outputs the available JMX metrics.

```bash
/opt/newrelic-infra/newrelic-integrations/bin/nri-jmx -query "*:*"
```

### 3.2 Creating the metric collection configuration file

In the JMX configuration file, we specified a collection file `jmx-custom-metrics.yml`. This file is used to define which metrics we want to collect.

We can inspect the available JMX metrics using nri-jmx command directly or a visual tool like JConsole.

```bash
/opt/newrelic-infra/newrelic-integrations/bin/nri-jmx -query "*:*"
```

or you can start with [template collectors file](../../tomcat-metrics.yml.sample)

### 3.3 Validate nri-jmx standalone

```/opt/newrelic-infra/newrelic-integrations/bin/nri-jmx -collection_files /etc/newrelic-infra/integrations.d/jmx-custom-metrics.yml -jmx_port 9010 -jmx_host IP_OF_CONTAINER```

### 3.3 Checking data

Save the changes in the yaml files, and [restart](https://docs.newrelic.com/docs/infrastructure/install-infrastructure-agent/manage-your-agent/start-stop-restart-infrastructure-agent) the agent. After a few minutes, go to New Relic and run the following [NRQL query](https://docs.newrelic.com/docs/query-data/nrql-new-relic-query-language):

```sql
FROM TomcatSample SELECT *
```
