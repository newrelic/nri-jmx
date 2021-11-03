#!/usr/bin/env bash

microk8s kubectl delete deployments infra-agent-bundle tomcat-deployment 2>/dev/null
microk8s kubectl delete configmap nri-integration-cfg 2>/dev/null

microk8s kubectl apply -f /home/vagrant/k8s/kube-metrics
microk8s kubectl apply -f /home/vagrant/k8s/config-map-nri-jmx.yml
microk8s kubectl apply -f /home/vagrant/k8s/config-map-nr-env.yml
microk8s kubectl apply -f /home/vagrant/k8s/deployment-tomcat.yml
microk8s kubectl apply -f /home/vagrant/k8s/deployment-infra-agent-bundle.yml
