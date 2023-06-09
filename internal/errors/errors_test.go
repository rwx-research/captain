package errors_test

import (
	"fmt"

	"github.com/rwx-research/captain-cli/internal/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {
	Describe("ConfigurationError", func() {
		It("behaves like an error", func() {
			err := errors.NewConfigurationError(fmt.Sprintf("some error %v", "some value"), "", "")
			Expect(err.Error()).To(Equal("some error some value"))
			Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("/errors_test.go"))

			configErr, ok := errors.AsConfigurationError(err)

			Expect(ok).To(Equal(true))
			Expect(configErr).To(Equal(errors.Unwrap(err)))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("ExecutionError", func() {
		It("behaves like an error", func() {
			err := errors.NewExecutionError(2, "some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))
			Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("/errors_test.go"))

			executionErr, ok := errors.AsExecutionError(err)

			Expect(ok).To(Equal(true))
			Expect(executionErr).To(Equal(errors.Unwrap(err)))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("InputError", func() {
		It("behaves like an error", func() {
			err := errors.NewInputError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))
			Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("/errors_test.go"))

			inputErr, ok := errors.AsInputError(err)

			Expect(ok).To(Equal(true))
			Expect(inputErr).To(Equal(errors.Unwrap(err)))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("InternalError", func() {
		It("behaves like an error", func() {
			err := errors.NewInternalError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))
			Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("/errors_test.go"))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(true))
			Expect(internalErr).To(Equal(errors.Unwrap(err)))

			systemErr, ok := errors.AsSystemError(err)

			Expect(ok).To(Equal(false))
			Expect(systemErr.E).To(BeNil())
		})
	})

	Describe("SystemError", func() {
		It("behaves like an error", func() {
			err := errors.NewSystemError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))
			Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("/errors_test.go"))

			systemErr, ok := errors.AsSystemError(err)

			Expect(ok).To(Equal(true))
			Expect(systemErr).To(Equal(errors.Unwrap(err)))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})
})
