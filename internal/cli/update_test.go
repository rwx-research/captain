package cli_test

import (
	"context"
	"os"
	"strings"

	"github.com/rwx-research/captain-cli/internal/backend/local"
	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Update Captain", func() {
	const (
		flakesPath      = "flakes"
		quarantinesPath = "quarantines"
		timingsPath     = "timings"
	)

	var (
		args     []string
		ctx      context.Context
		mockedFS *mocks.FileSystem
		service  cli.Service

		flakes, quarantines, timings *mocks.File
	)

	BeforeEach(func() {
		args = []string{}

		flakes = &mocks.File{
			Builder: new(strings.Builder),
			Reader:  strings.NewReader(""),
		}
		quarantines = &mocks.File{
			Builder: new(strings.Builder),
			Reader:  strings.NewReader(""),
		}
		timings = &mocks.File{
			Builder: new(strings.Builder),
			Reader:  strings.NewReader(""),
		}

		mockedFS = new(mocks.FileSystem)
		mockedFS.MockOpen = func(name string) (fs.File, error) {
			switch name {
			case flakesPath:
				return flakes, nil
			case quarantinesPath:
				return quarantines, nil
			case timingsPath:
				return timings, nil
			default:
				return nil, errors.NewInternalError("unknown file")
			}
		}
		mockedFS.MockOpenFile = func(name string, _ int, _ os.FileMode) (fs.File, error) {
			return mockedFS.MockOpen(name)
		}
	})

	JustBeforeEach(func() {
		api, err := local.NewClient(mockedFS, flakesPath, quarantinesPath, timingsPath)
		Expect(err).NotTo(HaveOccurred())

		service = cli.Service{
			API:         api,
			FileSystem:  mockedFS,
			TaskRunner:  new(mocks.TaskRunner),
			ParseConfig: parsing.Config{},
		}
	})

	Describe("adding", func() {
		Context("a flake", func() {
			JustBeforeEach(func() {
				Expect(service.AddFlake(ctx, args)).To(Succeed())
			})

			Context("using some args", func() {
				BeforeEach(func() {
					args = []string{"--description", "my test", "--file", "test.js"}
				})

				It("updates the local flakes file", func() {
					fileContent := flakes.Builder.String()
					Expect(fileContent).To(HavePrefix("- description:"), "The first line should be the first argument")
					Expect(fileContent).To(ContainSubstring("description: my test"))
					Expect(fileContent).To(ContainSubstring("file: test.js"))
				})
			})
		})

		Context("a quarantined test", func() {
			JustBeforeEach(func() {
				Expect(service.AddQuarantine(ctx, args)).To(Succeed())
			})

			Context("using some args", func() {
				BeforeEach(func() {
					args = []string{"--project", "foobar", "--description", "another test"}
				})

				It("updates the local quarantines file", func() {
					fileContent := quarantines.Builder.String()
					Expect(fileContent).To(HavePrefix("- project:"), "The first line should be the first argument")
					Expect(fileContent).To(ContainSubstring("project: foobar"))
					Expect(fileContent).To(ContainSubstring("description: another test"))
				})
			})
		})
	})

	Describe("removing", func() {
		Context("a flake", func() {
			JustBeforeEach(func() {
				Expect(service.RemoveFlake(ctx, args)).To(Succeed())
			})

			Context("using some args", func() {
				BeforeEach(func() {
					args = []string{"--name", "test-1"}
					flakes.Reader = strings.NewReader("- name: test-1\n- name: test-2")
				})

				It("updates the local flakes file", func() {
					Expect(flakes.Builder.String()).To(Equal("- name: test-2\n"))
				})
			})
		})

		Context("a quarantined test", func() {
			JustBeforeEach(func() {
				Expect(service.RemoveQuarantine(ctx, args)).To(Succeed())
			})

			Context("using some args", func() {
				BeforeEach(func() {
					args = []string{"--description", "my test"}
					quarantines.Reader = strings.NewReader("- name: test-1\n- description: my test")
				})

				It("updates the local flakes file", func() {
					Expect(quarantines.Builder.String()).To(Equal("- name: test-1\n"))
				})
			})
		})
	})
})
