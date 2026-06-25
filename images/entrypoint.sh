#!/bin/sh

set -e

SRIOV_DP_SYS_BINARY_DIR="/usr/bin/"
CLI_PARAMS=""
LOG_DIR=""
LOG_LEVEL=10
RESOURCE_PREFIX=""
CONFIG_FILE=""
USE_CDI=false
LOG_MAX_SIZE=""
LOG_MAX_FILES=""
LOG_MAX_AGE=""

usage()
{
    /bin/echo -e "This is an entrypoint script for SR-IOV Network Device Plugin"
    /bin/echo -e ""
    /bin/echo -e "./entrypoint.sh"
    /bin/echo -e "\t-h --help"
    /bin/echo -e "\t--log-dir=$LOG_DIR"
    /bin/echo -e "\t--log-level=$LOG_LEVEL"
    /bin/echo -e "\t--log-max-size=<MB> (default: 100)"
    /bin/echo -e "\t--log-max-files=<count> (default: 5)"
    /bin/echo -e "\t--log-max-age=<days> (default: 30)"
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
        --log-max-size)
            LOG_MAX_SIZE=$VALUE
            ;;
        --log-max-files)
            LOG_MAX_FILES=$VALUE
            ;;
        --log-max-age)
            LOG_MAX_AGE=$VALUE
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

if [ "$LOG_DIR" != "" ]; then
    # Support both relative (subdirectory under /var/log/) and absolute paths
    case "$LOG_DIR" in
        /*) LOG_PATH="$LOG_DIR" ;;
        *)  LOG_PATH="/var/log/$LOG_DIR" ;;
    esac
    mkdir -p "$LOG_PATH" || true
fi
CLI_PARAMS="-v $LOG_LEVEL"

if [ "$LOG_DIR" != "" ]; then
    CLI_PARAMS="$CLI_PARAMS --logtostderr --log_dir $LOG_PATH"
else
    CLI_PARAMS="$CLI_PARAMS --logtostderr"
fi

for spec in \
    "--log-max-size:$LOG_MAX_SIZE" \
    "--log-max-files:$LOG_MAX_FILES" \
    "--log-max-age:$LOG_MAX_AGE"
do
    flag_name=${spec%%:*}
    flag_value=${spec#*:}
    if [ "$flag_value" != "" ]; then
        CLI_PARAMS="$CLI_PARAMS $flag_name $flag_value"
    fi
done

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
