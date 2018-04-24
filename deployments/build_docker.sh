#!/usr/bin/env bash
set -e

## Build docker image
yes | cp ../bin/sriovdp .
# cd deployments
docker build -t sriov-device-plugin . 


