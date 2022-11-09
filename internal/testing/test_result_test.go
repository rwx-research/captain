package testing_test

import (
	"time"

	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestResult.Identify", func() {
	Context("with strict identification", func() {
		It("returns an error when fetching from meta of a test result without meta", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Duration:    time.Duration(12141040140104),
				Status:      testing.TestStatusFailed,
			}.Identify([]string{"description", "something_from_meta"}, true)

			Expect(compositeIdentifier).To(Equal(""))
			Expect(err.Error()).To(MatchRegexp("Meta is not defined"))
		})

		It("returns an error when fetching from meta of a test result without the component in meta", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Meta:        map[string]any{"something_else_in_meta": 1},
			}.Identify([]string{"description", "something_from_meta"}, true)

			Expect(compositeIdentifier).To(Equal(""))
			Expect(err.Error()).To(MatchRegexp("it was not there"))
		})

		It("returns a composite identifier otherwise", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Meta: map[string]any{
					"foo": 1,
					"bar": "hello",
					"baz": false,
					"nil": nil,
				},
			}.Identify([]string{"description", "foo", "bar", "baz", "nil"}, true)

			Expect(compositeIdentifier).To(Equal("the-description -captain- 1 -captain- hello -captain- false -captain- "))
			Expect(err).To(BeNil())
		})
	})

	Context("without strict identification", func() {
		It("returns a composite identifier when fetching from meta of a test result without meta", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Duration:    time.Duration(12141040140104),
				Status:      testing.TestStatusFailed,
			}.Identify([]string{"description", "something_from_meta"}, false)

			Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
			Expect(err).To(BeNil())
		})

		It("returns a composite identifier when fetching from meta of a test result without the component in meta", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Meta:        map[string]any{"something_else_in_meta": 1},
			}.Identify([]string{"description", "something_from_meta"}, false)

			Expect(compositeIdentifier).To(Equal("the-description -captain- MISSING_IDENTITY_COMPONENT"))
			Expect(err).To(BeNil())
		})

		It("returns a composite identifier otherwise", func() {
			compositeIdentifier, err := testing.TestResult{
				Description: "the-description",
				Meta: map[string]any{
					"foo": 1,
					"bar": "hello",
					"baz": false,
					"nil": nil,
				},
			}.Identify([]string{"description", "foo", "bar", "baz", "nil"}, false)

			Expect(compositeIdentifier).To(Equal("the-description -captain- 1 -captain- hello -captain- false -captain- "))
			Expect(err).To(BeNil())
		})
	})
})
