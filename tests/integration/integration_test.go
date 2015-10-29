package integration_test

import (
	. "github.com/bobtfish/AWSnycast/tests/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("Integration", func() {
	if testing.Short() {
		Skip("skipping test in short mode.")
	}
	var internalIPs []string
	BeforeEach(func() {
		RunMake()
		RunTerraform()
		internalIPs = InternalIPs()
	})
	Describe("Basic NAT machine tests", func() {
		Context("A availability zone", func() {
			It("should be able to ping 8.8.8.8", func() {
				Ssh("ping -c 2 8.8.8.8", NatA())
			})
		})
		Context("B availability zone", func() {
			It("should be able to ping 8.8.8.8", func() {
				Ssh("ping -c 2 8.8.8.8", NatB())
			})
		})
	})
	Describe("NAT works from inside", func() {
		for _, ip := range internalIPs {
			Context(ip, func() {
				It("should be able to ping 8.8.8.8", func() {
					out := Ssh("nc "+ip+" 8732", NatA())
					Ω(out).Should(ContainSubstring("64 bytes from 8.8.8.8"))
				})
			})
		}
	})
})
