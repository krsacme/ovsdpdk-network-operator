#!/bin/bash

if [ -z "$PKGS" ]; then
  # by default, test everything that's not in vendor
  PKGS="$(go list ./... | grep -v vendor | xargs echo)"
fi

GINKGO=`which ginkgo`
if [ $? != 0 ]; then
    echo "Download ginkgo..."
    go get github.com/onsi/ginkgo
fi

# Get only the package with test files
TESTPKGS=$(go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' ${PKGS})
PATHS=""
for item in ${TESTPKGS}; do
    PATHS+="${GOPATH}/src/${item} "
done
ginkgo ${PATHS}
retcode=$?

exit $retcode
