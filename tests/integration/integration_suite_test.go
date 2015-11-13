package integration_test

import (
	. "github.com/bobtfish/AWSnycast/tests/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var internalIPs []string

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	RegisterFailHandler(Fail)

	//RunMake()
	//RunTerraform()
	internalIPs = InternalIPs()
	RunSpecs(t, "Integration Suite")
}
