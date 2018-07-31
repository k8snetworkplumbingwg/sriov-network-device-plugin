#!/bin/bash

set -e

SRIOV_DP_BINARY_FILE="/usr/src/sriov-network-device-plugin/bin/sriovdp"
SYS_BINARY_DIR="/usr/bin/"

cp -f $SRIOV_DP_BINARY_FILE $SYS_BINARY_DIR

$SYS_BINARY_DIR/sriovdp --logtostderr -v 10
