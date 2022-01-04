#!/usr/bin/env bash

apt-get update

# install docker
which docker
if [ $? -ne 0 ]; then
  curl -fsSL get.docker.com -o get-docker.sh && sh get-docker.sh
fi

# install https://microk8s.io/
which microk8s || snap install microk8s --classic
# install DNS
microk8s enable dns helm helm3

# alias for convenience
grep "alias kubectl" /root/.bashrc || echo "alias kubectl='microk8s kubectl'" >>/root/.bashrc
