package errors_test

import (
	"github.com/rwx-research/captain-cli/internal/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {
	Describe("ConfigurationError", func() {
		It("behaves like an error", func() {
			err := errors.NewConfigurationError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))

			configErr, ok := errors.AsConfigurationError(err)

			Expect(ok).To(Equal(true))
			Expect(configErr).To(Equal(err))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("ExecutionError", func() {
		It("behaves like an error", func() {
			err := errors.NewExecutionError(2, "some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))

			executionErr, ok := errors.AsExecutionError(err)

			Expect(ok).To(Equal(true))
			Expect(executionErr).To(Equal(err))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("InputError", func() {
		It("behaves like an error", func() {
			err := errors.NewInputError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))

			inputErr, ok := errors.AsInputError(err)

			Expect(ok).To(Equal(true))
			Expect(inputErr).To(Equal(err))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})

	Describe("InternalError", func() {
		It("behaves like an error", func() {
			err := errors.NewInternalError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(true))
			Expect(internalErr).To(Equal(err))

			systemErr, ok := errors.AsSystemError(err)

			Expect(ok).To(Equal(false))
			Expect(systemErr.E).To(BeNil())
		})
	})

	Describe("SystemError", func() {
		It("behaves like an error", func() {
			err := errors.NewSystemError("some error %v", "some value")
			Expect(err.Error()).To(Equal("some error some value"))

			systemErr, ok := errors.AsSystemError(err)

			Expect(ok).To(Equal(true))
			Expect(systemErr).To(Equal(err))

			internalErr, ok := errors.AsInternalError(err)

			Expect(ok).To(Equal(false))
			Expect(internalErr.E).To(BeNil())
		})
	})
})
