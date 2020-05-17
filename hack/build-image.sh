#!/usr/bin/env bash

set -eu

DOCKER_PREFIX=${DOCKER_PREFIX:-quay.io/krsacme}
DOCKER_TAG=${DOCKER_TAG:=latest}
CONTAINER_CLI=podman

DEV=${DEV:-false}

if ! which $CONTAINER_CLI>/dev/null; then
    yum install -y podman
fi

DOCKERFILE="Dockerfile"
if [[ $DEV == "true" ]]; then
    DOCKERFILE="Dockerfile.dev"
fi

BIN_MD5_NEW=$(find bindata/ -type f -exec md5sum {} \; | md5sum | cut -d ' ' -f1)
BIN_MD5="_output/bindata.md5"
CLEAR=1
if [[ -f $BIN_MD5 ]]; then 
    BIN_MD5_OLD=`cat $BIN_MD5`
    if [[ $BIN_MD5_OLD == $BIN_MD5_NEW ]]; then
        CLEAR=0
    fi
fi
echo $BIN_MD5_NEW > $BIN_MD5
if [[ $CLEAR == 1 ]]; then
    set +e
    NAME="ovsdpdk-network-operator"
    rm _output/ovsdpdk-network-operator.md5
    $CONTAINER_CLI rmi -f ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
    set -e
fi

IMAGES="operator prepare"
for i in $IMAGES; do
    NAME="ovsdpdk-network-$i"
    EXE="_output/$NAME"
    MD5="$EXE.md5"
    if [[ ! -f $EXE ]]; then
        echo "Executable not found ($EXE)"
        exit 1
    fi

    NEW=$(find bindata/ -type f -exec md5sum {} \; | md5sum $EXE | md5sum | cut -d' ' -f1)
    if [[ -f $MD5 ]]; then
        OLD=`cat $MD5`
        if [[ $NEW == $OLD ]]; then
            echo "Build skipped, no change in $NAME..."
            continue
        fi
    fi
    echo $NEW > $MD5

    echo "Building container image $NAME ..."
    $CONTAINER_CLI build -f build/$i/${DOCKERFILE} . -t ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
    if [[ $DEV == "true" ]]; then
        $CONTAINER_CLI push ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
    fi
done

NAME=userspace-cni
$CONTAINER_CLI build -f build/userspace-cni/Dockerfile . -t ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
if [[ $DEV == "true" ]]; then
    $CONTAINER_CLI push ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
fi
