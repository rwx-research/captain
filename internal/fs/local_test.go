package fs_test

import (
	"os"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fs.GlobMany", func() {
	It("expands a single glob pattern", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{"../../test/fixtures/integration-tests/partition/*_spec.rb"})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/a_spec.rb",
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/c_spec.rb",
			"../../test/fixtures/integration-tests/partition/d_spec.rb",
		}))
	})

	It("expands multiple glob patterns", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../test/fixtures/integration-tests/partition/*_spec.rb",
			"../../test/fixtures/integration-tests/partition/x.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/a_spec.rb",
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/c_spec.rb",
			"../../test/fixtures/integration-tests/partition/d_spec.rb",
			"../../test/fixtures/integration-tests/partition/x.rb",
		}))
	})

	It("supports negation", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../test/fixtures/integration-tests/partition/*_spec.rb",
			"!../../test/fixtures/integration-tests/partition/a_spec.rb",
			"!../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/c_spec.rb",
			"../../test/fixtures/integration-tests/partition/d_spec.rb",
		}))
	})

	It("expands multiple glob patterns only returning unique paths", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../test/fixtures/integration-tests/partition/*_spec.rb",
			"../../test/fixtures/integration-tests/partition/*_spec.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/a_spec.rb",
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/c_spec.rb",
			"../../test/fixtures/integration-tests/partition/d_spec.rb",
		}))
	})

	It("sorts the results for determinism", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../test/fixtures/integration-tests/partition/z.rb",
			"../../test/fixtures/integration-tests/partition/y.rb",
			"../../test/fixtures/integration-tests/partition/x.rb",
			"../../test/fixtures/integration-tests/partition/x.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/x.rb",
			"../../test/fixtures/integration-tests/partition/y.rb",
			"../../test/fixtures/integration-tests/partition/z.rb",
		}))
	})

	It("recursively expands double star globs", func() {
		fs := fs.Local{}
		expandedPaths, _ := fs.GlobMany([]string{
			"../../test/**/*.rb",
		})

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/integration-tests/partition/a_spec.rb",
			"../../test/fixtures/integration-tests/partition/b_spec.rb",
			"../../test/fixtures/integration-tests/partition/c_spec.rb",
			"../../test/fixtures/integration-tests/partition/d_spec.rb",
			"../../test/fixtures/integration-tests/partition/x.rb",
			"../../test/fixtures/integration-tests/partition/y.rb",
			"../../test/fixtures/integration-tests/partition/z.rb",
		}))
	})

	It("supports files with special characters", func() {
		fs := fs.Local{}
		expandedPaths, err := fs.GlobMany([]string{
			"../../test/fixtures/filenames/**/*.txt",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/filenames/nested/$ @=:+{}[]^><~#|.txt",
			"../../test/fixtures/filenames/nested/**.txt",
			"../../test/fixtures/filenames/nested/*.txt",
			"../../test/fixtures/filenames/nested/?.txt",
			"../../test/fixtures/filenames/nested/[].txt",
			"../../test/fixtures/filenames/nested/\\.txt",
		}))
	})

	It("supports escaping *s", func() {
		fs := fs.Local{}
		expandedPaths, err := fs.GlobMany([]string{
			"../../test/fixtures/filenames/**/\\*.txt",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/filenames/nested/*.txt",
		}))
	})

	It("supports escaping **s", func() {
		fs := fs.Local{}
		expandedPaths, err := fs.GlobMany([]string{
			"../../test/fixtures/filenames/**/\\*\\*.txt",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(expandedPaths).To(Equal([]string{
			"../../test/fixtures/filenames/nested/**.txt",
		}))
	})
})

var _ = Describe("fs.Stat", func() {
	It("returns info on a file that exists", func() {
		fs := fs.Local{}
		info, err := fs.Stat("../../go.mod")

		Expect(err).NotTo(HaveOccurred())
		Expect(info).NotTo(BeNil())
	})

	It("errs on a file that does not exist", func() {
		fs := fs.Local{}
		info, err := fs.Stat("../../does-not-exist")

		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
		Expect(info).To(BeNil())
	})
})

var _ = Describe("fs.CreateTemp", func() {
	It("creates a temporary file", func() {
		fs := fs.Local{}
		file, err := fs.CreateTemp("", "create-temp-file")
		defer os.Remove(file.Name())

		Expect(err).NotTo(HaveOccurred())
		Expect(file.Name()).To(ContainSubstring("create-temp-file"))
	})
})
