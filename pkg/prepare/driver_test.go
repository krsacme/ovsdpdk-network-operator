package prepare_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	iowrap "github.com/spf13/afero"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdk/v1"
	. "github.com/krsacme/ovsdpdk-network-operator/pkg/prepare"
)

func init() {
	FS = iowrap.NewMemMapFs()
	FSUtil = &iowrap.Afero{Fs: FS}
}

var _ = Describe("Pkg/Prepare/Interface", func() {

	Context("Invalid interface", func() {
	})
	Context("Invalid PCI address", func() {
	})
})
