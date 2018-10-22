#!/bin/bash

set -e

SRIOV_DP_BINARY_FILE="/usr/src/sriov-network-device-plugin/build/sriovdp"
SRIOV_DP_SYS_BINARY_DIR="/usr/bin/"
LOG_DIR=""
LOG_LEVEL=10

function usage()
{
    echo -e "This is an entrypoint script for SR-IOV Network Device Plugin"
    echo -e ""
    echo -e "./entrypoint.sh"
    echo -e "\t-h --help"
    echo -e "\t--log-dir=$LOG_DIR"
    echo -e "\t--log-level=$LOG_LEVEL"
}

while [ "$1" != "" ]; do
    PARAM=`echo $1 | awk -F= '{print $1}'`
    VALUE=`echo $1 | awk -F= '{print $2}'`
    case $PARAM in
        -h | --help)
            usage
            exit
            ;;
        --log-dir)
            LOG_DIR=$VALUE
            ;;
        --log-level)
            LOG_LEVEL=$VALUE
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done

cp -f $SRIOV_DP_BINARY_FILE $SRIOV_DP_SYS_BINARY_DIR

if [ "$LOG_DIR" != "" ]; then
    mkdir -p "/var/log/$LOG_DIR"
    $SRIOV_DP_SYS_BINARY_DIR/sriovdp --log_dir "/var/log/$LOG_DIR" --alsologtostderr -v $LOG_LEVEL
else
    $SRIOV_DP_SYS_BINARY_DIR/sriovdp --logtostderr -v $LOG_LEVEL
fi
