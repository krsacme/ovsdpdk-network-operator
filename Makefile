# Copied from coreos-assembler
GOARCH := $(shell uname -m)
ifeq ($(GOARCH),x86_64)
	GOARCH = amd64
else ifeq ($(GOARCH),aarch64)
	GOARCH = arm64
endif

# vim: noexpandtab ts=8
export GOPATH=$(shell echo $${GOPATH:-$$HOME/go})
export GO111MODULE
export GOPROXY=https://proxy.golang.org

export OVSDPDK_GO_PACKAGE=github.com/krsacme/ovsdpdk-network-operator

#### TARGETS
default: build

operator-sdk:
	@if ! type -p operator-sdk ; \
	then if [ ! -d $(GOPATH)/src/github.com/operator-framework/operator-sdk ] ; \
	  then git clone https://github.com/operator-framework/operator-sdk --branch master $(GOPATH)/src/github.com/operator-framework/operator-sdk ; \
	  fi ; \
	  cd $(GOPATH)/src/github.com/operator-framework/operator-sdk ; \
	  make dep ; \
	  make install ; \
	fi

generate: operator-sdk
	hack/update-codegen.sh

build: ovsdpdk-network-operator ovsdpdk-network-prepare

ovsdpdk-network-operator:
	WHAT=ovsdpdk-network-operator hack/build-go.sh

ovsdpdk-network-prepare:
	WHAT=ovsdpdk-network-prepare hack/build-go.sh

image:
	hack/build-image.sh

dev image-dev: build
	DEV=true hack/build-image.sh

check test:
	hack/test-go.sh ${PKGS}

.PHONY: clean
clean:
	@rm -rf _output
