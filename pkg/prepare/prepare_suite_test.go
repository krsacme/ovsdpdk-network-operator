package prepare_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOvsDpdkPrepare(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OvSDPDK Prepare Suite")
}
