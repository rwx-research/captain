package fs_test

import (
	"github.com/rwx-research/captain-cli/internal/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fs.GlobMany", func() {
	It("expands a single glob pattern", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{"../../.github/workflows/fixtures/partition/*_spec.rb"})

		Expect(expandedPaths).To(Equal([]string{
			"../../.github/workflows/fixtures/partition/a_spec.rb",
			"../../.github/workflows/fixtures/partition/b_spec.rb",
			"../../.github/workflows/fixtures/partition/c_spec.rb",
			"../../.github/workflows/fixtures/partition/d_spec.rb",
		}))
	})

	It("expands multiple glob patterns", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../.github/workflows/fixtures/partition/*_spec.rb",
			"../../.github/workflows/fixtures/partition/x.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../.github/workflows/fixtures/partition/a_spec.rb",
			"../../.github/workflows/fixtures/partition/b_spec.rb",
			"../../.github/workflows/fixtures/partition/c_spec.rb",
			"../../.github/workflows/fixtures/partition/d_spec.rb",
			"../../.github/workflows/fixtures/partition/x.rb",
		}))
	})

	It("expands multiple glob patterns only returning unique paths", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../.github/workflows/fixtures/partition/*_spec.rb",
			"../../.github/workflows/fixtures/partition/*_spec.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../.github/workflows/fixtures/partition/a_spec.rb",
			"../../.github/workflows/fixtures/partition/b_spec.rb",
			"../../.github/workflows/fixtures/partition/c_spec.rb",
			"../../.github/workflows/fixtures/partition/d_spec.rb",
		}))
	})

	It("sorts the results for determinism", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../.github/workflows/fixtures/partition/z.rb",
			"../../.github/workflows/fixtures/partition/y.rb",
			"../../.github/workflows/fixtures/partition/x.rb",
			"../../.github/workflows/fixtures/partition/x.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../.github/workflows/fixtures/partition/x.rb",
			"../../.github/workflows/fixtures/partition/y.rb",
			"../../.github/workflows/fixtures/partition/z.rb",
		}))
	})
})
