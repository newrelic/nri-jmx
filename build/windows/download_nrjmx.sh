#!/usr/bin/env bash
set -e

# Gets last version of nrjmx MSI installer

echo "Downlading last version of nrjmx MSI installer"
if [[ -z $NRJMX_URL ]]; then
  echo "Generating nrjmx asset url"
  if [[ -z $NRJMX_VERSION ]]; then
    echo "Fetching latest nrjmx version"
    NRJMX_VERSION=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/tags"| grep 'name' | grep -oE '[0-9.?]+' | sort -V | tail -n 1)
    echo $NRJMX_VERSION
  fi
  NRJMX_VERSION="2.10.1"
  echo "Using latest nrjmx version $NRJMX_VERSION."
  NRJMX_URL=https://github.com/newrelic/nrjmx/releases/download/v$NRJMX_VERSION/nrjmx-amd64.$NRJMX_VERSION.msi
  echo $NRJMX_URL
fi

curl -L -Ss --fail "$NRJMX_URL" -o "build/package/windows/bundle/nrjmx-amd64.msi"
