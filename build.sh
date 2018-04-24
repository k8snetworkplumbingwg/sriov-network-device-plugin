#!/usr/bin/env bash
set -e
# Copyright 2018 Intel Corp. All Rights Reserved.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ "$(uname)" == "Darwin" ]; then
	export GOOS=linux
fi

PROJ="sriov-network-device-plugin"
ORG_PATH="github.com/intel"
REPO_PATH="${ORG_PATH}/${PROJ}"

if [ ! -h gopath/src/${REPO_PATH} ]; then
	mkdir -p gopath/src/${ORG_PATH}
	ln -s ../../../.. gopath/src/${REPO_PATH} || exit 255
fi

export GO15VENDOREXPERIMENT=1
export GOPATH=${PWD}/gopath
export GO="${GO:-go}"

mkdir -p "${PWD}/bin"
export GOBIN=${PWD}/bin


echo "Building protobuf API"
protoc -I api/ -I${GOPATH}/src api/api.proto --go_out=plugins=grpc:api

echo "Building SRIOV Device plugin"
$GO install "$@" ${REPO_PATH}/cmd/sriovdp

echo "Building CNI-Shim"
$GO install "$@" ${REPO_PATH}/cmd/cnishim

## Build docker image
# yes | cp bin/sriov_dp deployments/
# cd deployments
# docker build -t sriov-device-plugin . 

echo "Done!"

