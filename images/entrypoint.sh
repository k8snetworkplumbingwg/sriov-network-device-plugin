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
    /bin/echo -e "\t--log_dir=/var/log/sriovdp"
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
        --log_dir)
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
    if printf '%s' "$LOG_DIR" | grep -qP '[\x00-\x1f\x7f]'; then
        echo "ERROR: --log_dir contains control characters" >&2; exit 1
    fi
    case "$LOG_DIR" in
        -*)
            echo "ERROR: --log_dir must not start with -" >&2; exit 1 ;;
    esac
    if printf '%s' "$LOG_DIR" | grep -q '[[:space:]]'; then
        echo "ERROR: --log_dir must not contain whitespace" >&2; exit 1
    fi
    case "$LOG_DIR" in
        *"'"*|*\"*)
            echo "ERROR: --log_dir must not contain quote characters" >&2; exit 1 ;;
    esac
    case "$LOG_DIR" in
        *[\*\?\`\$\|\&\;\!\(\)\{\}\[\]]*)
            echo "ERROR: --log_dir contains forbidden shell metacharacters" >&2; exit 1 ;;
    esac
    case "$LOG_DIR" in
        *..*)
            echo "ERROR: --log_dir must not contain '..'" >&2; exit 1 ;;
    esac
    case "$LOG_DIR" in
        *~*)
            echo "ERROR: --log_dir must not contain '~'" >&2; exit 1 ;;
    esac
    case "$LOG_DIR" in
        /var/log|/var/log/*)
            ;;
        *)
            echo "ERROR: --log_dir must be an absolute path under /var/log (got: $LOG_DIR)" >&2
            exit 1
            ;;
    esac

    mkdir -p "$LOG_DIR" || { echo "ERROR: cannot create log directory $LOG_DIR" >&2; exit 1; }

    # Resolve symlinks and confirm the real path is still under /var/log.
    REAL_LOG_PATH="$(cd "$LOG_DIR" && pwd -P)"
    case "$REAL_LOG_PATH" in
        /var/log|/var/log/*)
            LOG_DIR="$REAL_LOG_PATH"
            ;;
        *)
            echo "ERROR: --log_dir resolves to $REAL_LOG_PATH which is outside /var/log/" >&2
            exit 1
            ;;
    esac
fi
CLI_PARAMS="-v $LOG_LEVEL"

if [ "$LOG_DIR" != "" ]; then
    CLI_PARAMS="$CLI_PARAMS --logtostderr --log_dir=$LOG_DIR"
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
