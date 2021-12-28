[Vagrant](https://www.vagrantup.com/) managed Virtual Machine with [k8s cluster](https://microk8s.io/) and [infrastructure-bundle](https://github.com/newrelic/infrastructure-bundle) 
configured to monitor java application using `nri-jmx` and k8s annotations.

# Run

* Install Vagrant: https://www.vagrantup.com/downloads
* Create `provision/create-secret.sh` and add NR license key to it:
```bash
    cp provision/create-secret.sh.dist provision/create-secret.sh
    # Edit provision/create-secret.sh and add NR license
```
* Define cluster name and environment
```bash
    cp k8s/config-map-nr-env.yml.dist k8s/config-map-nr-env.yml
    # Edit k8s/config-map-nr-env.yml and set cluster name and environment
```

* Spawn VM:
```bash
    vagrant up
```

* Check metrics:
```
FROM TomcatSample SELECT * SINCE 5 minutes ago
```

# K8s
* [configMap for jmx](./k8s/config-map-nri-jmx.yml) : configMap for `nri-jmx` configuration and annotations discovery command.
* [infra-bundle deployment](./k8s/deployment-infra-agent-bundle.yml) : [infrastructure-bundle](https://github.com/newrelic/infrastructure-bundle) deployment.
* [Tomcat deployment](./k8s/deployment-tomcat.yml) : Example of Java application (Tomcat) with nri-jmx collection configuration using annotations.