#!/bin/sh

set -e

SRIOV_DP_SYS_BINARY_DIR="/usr/bin/"
LOG_DIR=""
LOG_LEVEL=10
RESOURCE_PREFIX=""
CONFIG_FILE=""
CLI_PARAMS=""
USE_CDI=false

usage()
{
    /bin/echo -e "This is an entrypoint script for SR-IOV Network Device Plugin"
    /bin/echo -e ""
    /bin/echo -e "./entrypoint.sh"
    /bin/echo -e "\t-h --help"
    /bin/echo -e "\t--log-dir=$LOG_DIR"
    /bin/echo -e "\t--log-level=$LOG_LEVEL"
    /bin/echo -e "\t--resource-prefix=$RESOURCE_PREFIX"
    /bin/echo -e "\t--config-file=$CONFIG_FILE"
    /bin/echo -e "\t--use-cdi"
}

while [ "$1" != "" ]; do
    PARAM="$(echo "$1" | awk -F= '{print $1}')"
    VALUE="$(echo "$1" | awk -F= '{print $2}')"
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
        --resource-prefix)
            RESOURCE_PREFIX=$VALUE
            ;;
        --config-file)
            CONFIG_FILE=$VALUE
            ;;
        --use-cdi)
            USE_CDI=true
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done

CLI_PARAMS="-v $LOG_LEVEL"

if [ "$LOG_DIR" != "" ]; then
    mkdir -p "/var/log/$LOG_DIR"
    CLI_PARAMS="$CLI_PARAMS --log_dir /var/log/$LOG_DIR --alsologtostderr"
else
    CLI_PARAMS="$CLI_PARAMS --logtostderr"
fi

if [ "$RESOURCE_PREFIX" != "" ]; then
    CLI_PARAMS="$CLI_PARAMS --resource-prefix $RESOURCE_PREFIX"
fi

if [ "$CONFIG_FILE" != "" ]; then
    CLI_PARAMS="$CLI_PARAMS --config-file $CONFIG_FILE"
fi

if [ "$USE_CDI" = true ]; then
    CLI_PARAMS="$CLI_PARAMS --use-cdi"
fi
set -f
# shellcheck disable=SC2086
exec $SRIOV_DP_SYS_BINARY_DIR/sriovdp $CLI_PARAMS
