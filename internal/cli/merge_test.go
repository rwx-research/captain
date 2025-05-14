package cli_test

import (
	"context"
	"io"
	"sort"
	"strings"

	"github.com/bradleyjkemp/cupaloy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/reporting"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type SuccessfulParser struct{}

func (p SuccessfulParser) Parse(_ io.Reader) (*v1.TestResults, error) {
	id := "test-id"
	one := "one"
	return &v1.TestResults{Summary: v1.Summary{Tests: 1}, Tests: []v1.Test{
		{
			ID:           &id,
			Name:         "test-name",
			Lineage:      []string{},
			Location:     &v1.Location{File: "test-file"},
			Attempt:      v1.TestAttempt{},
			PastAttempts: []v1.TestAttempt{},
		},
	}, Framework: v1.NewOtherFramework(&one, &one)}, nil
}

var _ = Describe("Merge", func() {
	var (
		ctx            context.Context
		service        cli.Service
		core           zapcore.Core
		mockFilesystem *mocks.FileSystem
	)

	BeforeEach(func() {
		core, _ = observer.New(zapcore.DebugLevel)
		mockFilesystem = new(mocks.FileSystem)
		logger := zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
			zap.WrapCore(func(_ zapcore.Core) zapcore.Core { return core }),
		)).Sugar()
		service = cli.Service{
			Log:        logger,
			FileSystem: mockFilesystem,
			ParseConfig: parsing.Config{
				Logger:                   logger,
				MutuallyExclusiveParsers: []parsing.Parser{},
				GenericParsers:           []parsing.Parser{SuccessfulParser{}},
			},
		}
	})

	Context("when given multiple globs", func() {
		It("globs each pattern and merges the results", func() {
			didGlobJSON := false
			didGlobXML := false

			mockGlob := func(pattern string) ([]string, error) {
				switch pattern {
				case "*.json":
					didGlobJSON = true
					return []string{"a.json", "b.json"}, nil
				case "*.xml":
					didGlobXML = true
					return []string{"a.xml", "b.xml"}, nil
				}

				return nil, nil
			}
			mockFilesystem.MockGlob = mockGlob

			openedFiles := []string{}

			mockOpen := func(name string) (fs.File, error) {
				openedFiles = append(openedFiles, name)

				return &mocks.File{
					Builder: new(strings.Builder),
					Reader:  strings.NewReader(""),
				}, nil
			}
			mockFilesystem.MockOpen = mockOpen

			mockCreate := func(_ string) (fs.File, error) {
				return &mocks.File{
					Builder: new(strings.Builder),
					Reader:  strings.NewReader(""),
				}, nil
			}
			mockFilesystem.MockCreate = mockCreate

			var reportedResults *v1.TestResults
			didReport := false
			err := service.Merge(ctx, cli.MergeConfig{
				ResultsGlobs: []string{"*.json", "*.xml"},
				Reporters: map[string]cli.Reporter{
					"test-reporter": func(_ fs.File, tr v1.TestResults, _ reporting.Configuration) error {
						didReport = true
						reportedResults = &tr
						return nil
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(didGlobJSON).To(BeTrue())
			Expect(didGlobXML).To(BeTrue())
			Expect(didReport).To(BeTrue())

			sort.Strings(openedFiles)
			Expect(openedFiles).To(Equal([]string{"a.json", "a.xml", "b.json", "b.xml"}))
			cupaloy.SnapshotT(GinkgoT(), *reportedResults)
		})
	})
})
