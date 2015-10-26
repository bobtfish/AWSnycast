package integration_test

import (
	. "github.com/bobtfish/AWSnycast/tests/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration", func() {
	BeforeEach(func() {
		RunMake()
		RunTerraform()
	})
	Describe("Categorizing book length", func() {
		Context("With more than 300 pages", func() {
			It("should be a novel", func() {
				Expect(Foo()).To(Equal(0))
			})
		})

	})

})
