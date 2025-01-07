package testresultsschema_test

import (
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Location", func() {
	Describe("String", func() {
		It("formats when there is a line number", func() {
			line := 15
			Expect(v1.Location{File: "some/file.rb", Line: &line}.String()).To(Equal("some/file.rb:15"))
		})

		It("formats when there is not a line number", func() {
			Expect(v1.Location{File: "some/file.rb"}.String()).To(Equal("some/file.rb"))
		})
	})
})
