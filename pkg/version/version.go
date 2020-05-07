package version

import (
	"fmt"

	"github.com/blang/semver"
)

var (
	// Raw is the string representation of the version. This will be replaced
	// with the calculated version at build time.
	Raw = "v1.0.0"

	// Version is semver representation of the version.
	Version = semver.MustParse("1.0.0")

	// String is the human-friendly representation of the version.
	String = fmt.Sprintf("ovsdpdk-network-operator %s", Raw)
)
