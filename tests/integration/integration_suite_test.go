package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
