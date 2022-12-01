package errors_test

import (
	"fmt"

	pkgerrors "github.com/pkg/errors"

	"github.com/rwx-research/captain-cli/internal/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("WithStack", func() {
		It("wraps an error without a message", func() {
			err := pkgerrors.New("some error")
			wrapped := errors.WithStack(err)
			Expect(wrapped.Error()).To(Equal("some error"))
			Expect(wrapped).NotTo(Equal(err))
			Expect(fmt.Sprintf("%+v", wrapped)).To(ContainSubstring("/utils_test.go"))

			var errPkg error
			ok := errors.As(err, &errPkg)

			Expect(ok).To(Equal(true))
			Expect(errPkg).To(Equal(err))
		})
	})

	Describe("Wrap", func() {
		It("wraps an error with a message", func() {
			err := pkgerrors.New("some error")
			wrapped := errors.Wrap(err, "some prefix")
			Expect(wrapped.Error()).To(Equal("some prefix: some error"))
			Expect(wrapped).NotTo(Equal(err))
			Expect(fmt.Sprintf("%+v", wrapped)).To(ContainSubstring("/utils_test.go"))

			var errPkg error
			ok := errors.As(err, &errPkg)

			Expect(ok).To(Equal(true))
			Expect(errPkg).To(Equal(err))
		})
	})

	Describe("Wrapf", func() {
		It("wraps an error with a formatted message", func() {
			err := pkgerrors.New("some error")
			wrapped := errors.Wrapf(err, "some prefix %v", "formatted")
			Expect(wrapped.Error()).To(Equal("some prefix formatted: some error"))
			Expect(wrapped).NotTo(Equal(err))
			Expect(fmt.Sprintf("%+v", wrapped)).To(ContainSubstring("/utils_test.go"))

			var errPkg error
			ok := errors.As(err, &errPkg)

			Expect(ok).To(Equal(true))
			Expect(errPkg).To(Equal(err))
		})
	})
})
