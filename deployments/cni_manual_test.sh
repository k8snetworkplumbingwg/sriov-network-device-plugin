#!/bin/bash

# TO-DO: Add netns if it not present

NETNS=blue
USAGE=$(cat <<- EOM
$0 -a | -d
        -a : CNI Add command
        -d : CNI Del command
EOM
)


# CNI env variables
export CNI_COMMAND=ADD
export CNI_CONTAINERID=${NETNS}
export CNI_NETNS=/var/run/netns/${NETNS}
export CNI_IFNAME=net1
export CNI_PATH=/opt/cni/bin 

# Take in CNI Command from argument
case "$1" in
    -a) CMD=ADD
        echo "Executing CNI Add"
        ip netns del ${NETNS}
        ip netns add ${NETNS}
    ;;
    -d) CMD=DEL
        echo "Executing CNI Del"
    ;;
    *) 
        echo "$USAGE"
        exit
    ;;
esac

export CNI_COMMAND=$CMD
/opt/cni/bin/sriov < ./sriov2-conf.json
echo "Done!"
