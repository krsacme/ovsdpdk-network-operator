#!/usr/bin/env bash

set -eu

NAME=ovsdpdk-network-operator
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

$CONTAINER_CLI build -f build/${DOCKERFILE} . -t ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}

if [[ $DEV == "true" ]]; then
    $CONTAINER_CLI push ${DOCKER_PREFIX}/${NAME}:${DOCKER_TAG}
fi
