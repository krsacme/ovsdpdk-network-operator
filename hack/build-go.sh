#!/usr/bin/env bash

set -eu

REPO=github.com/krsacme/ovsdpdk-network-operator
WHAT=${WHAT:-ovsdpdk-network-operator}
GOTAGS=${GOTAGS:-}
GLDFLAGS=${GLDFLAGS:-}

eval $(go env | grep -e "GOHOSTOS" -e "GOHOSTARCH")
if [ -z "${GOHOSTOS:-}" ] || [ -z "${GOHOSTARCH:-}" ]; then
  go env
  echo 'Failed to find expected variables in `go env` output above' 1>&2
  exit 1
fi

: "${GOOS:=${GOHOSTOS}}"
: "${GOARCH:=${GOHOSTARCH}}"

# Go to the root of the repo
cdup="$(git rev-parse --show-cdup)" && test -n "$cdup" && cd "$cdup"

if [ -z ${VERSION_OVERRIDE+a} ]; then
	echo "Using version from git..."
	VERSION_OVERRIDE=$(git describe --abbrev=8 --dirty --always)
fi

HASH=${SOURCE_GIT_COMMIT:-$(git rev-parse --verify 'HEAD^{commit}')}

GLDFLAGS+="-X ${REPO}/pkg/version.Raw=${VERSION_OVERRIDE} -X ${REPO}/pkg/version.Hash=${HASH}"

if [ -z ${BIN_PATH+a} ]; then
	export BIN_PATH=_output
fi

mkdir -p ${BIN_PATH}

CGO_ENABLED=0

echo "Building ${REPO}/cmd/${WHAT} (${VERSION_OVERRIDE}, ${HASH})"
#CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -mod=vendor -tags="${GOTAGS}" -ldflags "${GLDFLAGS} -s -w" -o ${BIN_PATH}/${WHAT} ${REPO}/cmd/${WHAT}
CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -tags="${GOTAGS}" -ldflags "${GLDFLAGS} -s -w" -o ${BIN_PATH}/${WHAT} ${REPO}/cmd/${WHAT}
