package targetedretries_test

import (
	"github.com/rwx-research/captain-cli/internal/targetedretries"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShellEscape", func() {
	It("ignores values without single quotes", func() {
		Expect(targetedretries.ShellEscape("some value")).To(Equal("some value"))
	})

	It("escapes single quotes, no matter where they are", func() {
		Expect(targetedretries.ShellEscape("'some ' val'ue'")).To(Equal(
			`'"'"'some '"'"' val'"'"'ue'"'"'`,
		))
	})
})

var _ = Describe("RegexpEscape", func() {
	It("escapes regexp meta characters", func() {
		Expect(targetedretries.RegexpEscape("a test with a + and a .")).To(Equal(
			`a test with a \+ and a \.`,
		))
	})
})
