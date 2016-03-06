package integration_test

import (
	. "AWSnycast/tests/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	xinetdNatCheckPort     = "8732"
	xinetdAnycastCheckPort = "8733"
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
				out := Ssh("nc "+internalIPs[0]+" "+xinetdNatCheckPort, NatA())
				Ω(out).Should(ContainSubstring("OK"))
			})
			It("should see an Anycast service (192.168.0.1)", func() {
				Ssh("nc "+internalIPs[0]+" "+xinetdAnycastCheckPort, NatA())
			})
			It("should see an Anycast service (192.168.0.1) in az a", func() {
				out := Ssh("nc "+internalIPs[0]+" "+xinetdAnycastCheckPort, NatA())
				Ω(out).Should(ContainSubstring("a"))
			})
		})
		Context("B", func() {
			It("should be able to ping 8.8.8.8", func() {
				out := Ssh("nc "+internalIPs[1]+" "+xinetdNatCheckPort, NatB())
				Ω(out).Should(ContainSubstring("OK"))
			})
			It("should see an Anycast service (192.168.0.1)", func() {
				Ssh("nc "+internalIPs[1]+" "+xinetdAnycastCheckPort, NatB())
			})
			It("should see an Anycast service (192.168.0.1) in az b", func() {
				out := Ssh("nc "+internalIPs[1]+" "+xinetdAnycastCheckPort, NatB())
				Ω(out).Should(ContainSubstring("b"))
			})
		})

	})
})
