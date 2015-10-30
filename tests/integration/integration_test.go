package integration_test

import (
	. "github.com/bobtfish/AWSnycast/tests/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration", func() {
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
		Context("Internal servers", func() {
			It("Shoud have 2 internal servers", func() {
				Ω(len(internalIPs)).Should(Equal(2))
			})
		})
	})
	Describe("Basic internal machine tests", func() {
		Context("A", func() {
			It("should be able to ping 8.8.8.8", func() {
				out := Ssh("nc "+internalIPs[0]+" 8732", NatA())
				Ω(out).Should(ContainSubstring("OK"))
			})
		})
		Context("B", func() {
			It("should be able to ping 8.8.8.8", func() {
				out := Ssh("nc "+internalIPs[1]+" 8732", NatB())
				Ω(out).Should(ContainSubstring("OK"))
			})
		})

	})
})
