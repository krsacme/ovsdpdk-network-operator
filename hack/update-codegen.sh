#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT="$(dirname "${BASH_SOURCE[0]}")/.."

# FIXME: code-generator is not created in vendor directory, fix it
#set +e
#go get -u k8s.io/code-generator
#set -e

GEN="${SCRIPT_ROOT}/vendor/k8s.io/code-generator/generate-groups.sh"
if [ ! -f $GEN ]; then
    if [ -f "$GOPATH/src/k8s.io/code-generator/generate-groups.sh" ]; then
        GEN="$GOPATH/src/k8s.io/code-generator/generate-groups.sh"
    fi
fi

$GEN deepcopy \
  github.com/krsacme/ovsdpdk-network-operator/pkg/generated github.com/krsacme/ovsdpdk-network-operator/pkg/apis \
  "ovsdpdknetwork:v1" \
  --go-header-file "${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt"

echo "Generating Operator Crds..."
operator-sdk generate crds

echo "Creating single deploy yaml file..."
files="service_account.yaml clusterrole.yaml clusterrole_binding.yaml role.yaml role_binding.yaml operator.yaml"
target="deploy/allinone.yaml"
>| deploy/allinone.yaml
for file in $files
do
    cat "deploy/files/$file" >> $target
    echo "---" >> $target
done
cat "deploy/crds/ovsdpdknetwork.openshift.io_ovsdpdkconfigs_crd.yaml" >> $target

