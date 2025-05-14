package cli_test

import (
	"context"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/config"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/mocks"
	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func cfgWithArgs(index int, total int, args []string, delimiter string, roundRobin bool, omitPrefix string) cli.PartitionConfig {
	return cli.PartitionConfig{
		TestFilePaths: args,
		PartitionNodes: config.PartitionNodes{
			Index: index,
			Total: total,
		},
		SuiteID:    "captain-cli-test",
		Delimiter:  delimiter,
		RoundRobin: roundRobin,
		OmitPrefix: omitPrefix,
	}
}

func cfgWithGlob(index int, total int, glob string) cli.PartitionConfig {
	return cfgWithArgs(index, total, []string{glob}, " ", false, "")
}

func cfgWithGlobAndRoundRobin(index int, total int, glob string) cli.PartitionConfig {
	return cfgWithArgs(index, total, []string{glob}, " ", true, "")
}

var _ = Describe("Partition", func() {
	var (
		err          error
		ctx          context.Context
		service      cli.Service
		core         zapcore.Core
		recordedLogs *observer.ObservedLogs

		fetchedTimingManifest bool
	)

	BeforeEach(func() {
		err = nil
		fetchedTimingManifest = false

		core, recordedLogs = observer.New(zapcore.DebugLevel)
		service = cli.Service{
			API: new(mocks.API),
			Log: zaptest.NewLogger(GinkgoT(), zaptest.WrapOptions(
				zap.WrapCore(func(_ zapcore.Core) zapcore.Core { return core }),
			)).Sugar(),
			FileSystem:  new(mocks.FileSystem),
			TaskRunner:  new(mocks.TaskRunner),
			ParseConfig: parsing.Config{},
		}
	})

	Context("when misconfigured", func() {
		It("requires an index be >= 0", func() {
			err = service.Partition(ctx, cfgWithGlob(-1, 2, "*.test"))
			Expect(err.Error()).To(ContainSubstring("Missing partition index"))
		})

		It("requires an index be < total", func() {
			err = service.Partition(ctx, cfgWithGlob(1, 1, "*.test"))
			Expect(err.Error()).To(ContainSubstring("Unsupported partitioning setup"))
			err = service.Partition(ctx, cfgWithGlob(2, 1, "*.test"))
			Expect(err.Error()).To(ContainSubstring("Unsupported partitioning setup"))
		})

		It("must specify filepath args", func() {
			err = service.Partition(ctx, cfgWithArgs(0, 1, []string{}, " ", false, ""))
			Expect(err.Error()).To(ContainSubstring("Missing test file paths"))
		})
	})

	Context("when the client provides multiple globs", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(_ context.Context, _ string) ([]testing.TestFileTiming, error) {
				return []testing.TestFileTiming{}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("only considers unique filepaths", func() {
			_ = service.Partition(ctx, cfgWithArgs(0, 1, []string{"*.test", "*.test"}, " ", false, ""))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test b.test c.test d.test"))
		})
	})

	Context("under expected conditions", func() {
		BeforeEach(func() {
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "a.test", Duration: 4},
					{Filepath: "b.test", Duration: 3},
					{Filepath: "c.test", Duration: 2},
					{Filepath: "d.test", Duration: 1},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("doesnt return an error", func() {
			err = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("fetches the timing manifest", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(fetchedTimingManifest).To(BeTrue())
		})

		It("uses least runtime strategy", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 10ns",
				"Target Partition Runtime: 5ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (3ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (2ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'd.test' (1ns) using least runtime strategy",
			}))
		})

		It("logs the partitioned files for index 0", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test d.test"))
		})

		It("logs the partitioned files for index 1", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test c.test"))
		})
	})

	Context("when round robin is specified", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("doesnt return an error", func() {
			err = service.Partition(ctx, cfgWithGlobAndRoundRobin(0, 2, "*.test"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("uses round-robin strategy", func() {
			_ = service.Partition(ctx, cfgWithGlobAndRoundRobin(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(Equal([]string{
				"Total Runtime: 0s",
				"Target Partition Runtime: 0s",
				"[PART 0 (0.00s)]: Assigned 'a.test' using round robin strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' using round robin strategy",
				"[PART 0 (0.00s)]: Assigned 'c.test' using round robin strategy",
				"[PART 1 (0.00s)]: Assigned 'd.test' using round robin strategy",
			}))
		})

		It("logs files for partition 0", func() {
			_ = service.Partition(ctx, cfgWithGlobAndRoundRobin(0, 2, "*.test"))

			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test c.test"))
		})

		It("logs files for partition 1", func() {
			_ = service.Partition(ctx, cfgWithGlobAndRoundRobin(1, 2, "*.test"))

			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test d.test"))
		})
	})

	Context("when there are no test file timings", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("prints warning to standard err", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.WarnLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(
				ContainElement("No test file timings were matched. Using naive round-robin strategy."),
			)
		})

		It("uses round-robin strategy", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(Equal([]string{
				"Total Runtime: 0s",
				"Target Partition Runtime: 0s",
				"[PART 0 (0.00s)]: Assigned 'a.test' using round robin strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' using round robin strategy",
				"[PART 0 (0.00s)]: Assigned 'c.test' using round robin strategy",
				"[PART 1 (0.00s)]: Assigned 'd.test' using round robin strategy",
			}))
		})

		It("logs files for partition 0", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))

			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test c.test"))
		})

		It("logs files for partition 1", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test d.test"))
		})
	})

	Context("when test file timings overflow partitions", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "a.test", Duration: 5},
					{Filepath: "b.test", Duration: 4},
					{Filepath: "c.test", Duration: 3},
					{Filepath: "d.test", Duration: 1},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("doesnt return an error", func() {
			err = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("fetches the timing manifest", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(fetchedTimingManifest).To(BeTrue())
		})

		It("uses least runtime strategy, falling back to most remaining when it can't fit", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 13ns",
				"Target Partition Runtime: 6ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (5ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (3ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'd.test' (1ns) using least runtime strategy",
			}))
		})

		It("logs the partitioned files for index 0", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test d.test"))
		})

		It("logs the partitioned files for index 1", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test c.test"))
		})
	})

	Context("when suite run with some timed and some untimed files", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "a.test", Duration: 6},
					{Filepath: "b.test", Duration: 4},
					{Filepath: "c.test", Duration: 3},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("doesnt return an error", func() {
			err = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("fetches the timing manifest", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			Expect(fetchedTimingManifest).To(BeTrue())
		})

		It("uses first fit, falls back to most remaining, then round robins unknowns", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 13ns",
				"Target Partition Runtime: 6ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (6ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (3ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'd.test' using round robin strategy",
			}))
		})

		It("logs the partitioned files for index 0", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test d.test"))
		})

		It("logs the partitioned files for index 1", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test c.test"))
		})
	})

	Context("when we're provided out of order timings", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "a.test", Duration: 1},
					{Filepath: "b.test", Duration: 2},
					{Filepath: "c.test", Duration: 3},
					{Filepath: "d.test", Duration: 4},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("sorts and processes in decreasing order", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 10ns",
				"Target Partition Runtime: 5ns",
				"[PART 0 (0.00s)]: Assigned 'd.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (3ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (2ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'a.test' (1ns) using least runtime strategy",
			}))
		})
	})

	Context("when we have moar partitions", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "d.test", Duration: 1},
					{Filepath: "c.test", Duration: 2},
					{Filepath: "b.test", Duration: 3},
					{Filepath: "a.test", Duration: 4},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("utilizes them best it can", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 3, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 10ns",
				"Target Partition Runtime: 3ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (3ns) using least runtime strategy",
				"[PART 2 (0.00s)]: Assigned 'c.test' (2ns) using least runtime strategy",
				"[PART 2 (0.00s)]: Assigned 'd.test' (1ns) using least runtime strategy",
			}))
		})
	})

	Context("when the server sends down ./filepaths", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				return []testing.TestFileTiming{
					{Filepath: "./d.test", Duration: 1},
					{Filepath: "./c.test", Duration: 2},
					{Filepath: "./b.test", Duration: 3},
					{Filepath: "./a.test", Duration: 4},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("still matches because we are comparing expanded absolute paths", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements([]string{
				"Total Runtime: 10ns",
				"Target Partition Runtime: 5ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (3ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (2ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'd.test' (1ns) using least runtime strategy",
			}))
		})
	})

	Context("when the server sends down fully expanded paths", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				a, _ := filepath.Abs("a.test")
				b, _ := filepath.Abs("b.test")
				c, _ := filepath.Abs("c.test")
				d, _ := filepath.Abs("d.test")
				return []testing.TestFileTiming{
					{Filepath: a, Duration: 4},
					{Filepath: b, Duration: 3},
					{Filepath: c, Duration: 2},
					{Filepath: d, Duration: 1},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("still matches because we are comparing expanded absolute paths", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements(
				"Total Runtime: 10ns",
				"Target Partition Runtime: 5ns",
				"[PART 0 (0.00s)]: Assigned 'a.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'b.test' (3ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'c.test' (2ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'd.test' (1ns) using least runtime strategy",
			))
		})

		It("logs the partitioned files for index 0 using client test file paths", func() {
			_ = service.Partition(ctx, cfgWithGlob(0, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test d.test"))
		})

		It("logs the partitioned files for index 1 using client test file paths", func() {
			_ = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("b.test c.test"))
		})
	})

	Context("when the server returns an error", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				return nil, errors.NewSystemError("something bad!")
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("raises an error because partitioning one index and round-robining another is problematic ", func() {
			err = service.Partition(ctx, cfgWithGlob(1, 2, "*.test"))
			Expect(err.Error()).To(ContainSubstring("something bad!"))
		})
	})

	Context("when using a custom delimiter", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"a.test", "b.test", "c.test", "d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				a, _ := filepath.Abs("a.test")
				b, _ := filepath.Abs("b.test")
				c, _ := filepath.Abs("c.test")
				d, _ := filepath.Abs("d.test")
				return []testing.TestFileTiming{
					{Filepath: a, Duration: 4},
					{Filepath: b, Duration: 3},
					{Filepath: c, Duration: 2},
					{Filepath: d, Duration: 1},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("raises an error because partitioning one index and round-robining another is problematic ", func() {
			_ = service.Partition(ctx, cfgWithArgs(0, 1, []string{"*.test"}, ",", false, ""))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("a.test,b.test,c.test,d.test"))
		})
	})

	Context("when using an omit prefix", func() {
		BeforeEach(func() {
			mockGlob := func(_ string) ([]string, error) {
				return []string{"test/a.test", "test/b.test", "test/c.test", "test/d.test"}, nil
			}
			mockGetTimingManifest := func(
				_ context.Context,
				_ string,
			) ([]testing.TestFileTiming, error) {
				fetchedTimingManifest = true
				a, _ := filepath.Abs("a.test")
				b, _ := filepath.Abs("b.test")
				c, _ := filepath.Abs("c.test")
				d, _ := filepath.Abs("d.test")
				return []testing.TestFileTiming{
					{Filepath: a, Duration: 4},
					{Filepath: b, Duration: 3},
					{Filepath: c, Duration: 2},
					{Filepath: d, Duration: 1},
				}, nil
			}
			service.API.(*mocks.API).MockGetTestTimingManifest = mockGetTimingManifest
			service.FileSystem.(*mocks.FileSystem).MockGlob = mockGlob
		})

		It("still matches because it first trims the omit-prefix", func() {
			globConfig := cfgWithArgs(1, 2, []string{"tests/*.test"}, " ", false, "test/")
			_ = service.Partition(ctx, globConfig)

			assignments := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.DebugLevel).All() {
				assignments = append(assignments, log.Message)
			}
			Expect(assignments).To(ContainElements(
				"Omitting prefix 'test/' from 'test/a.test' resulting in 'a.test' for comparison",
				"Omitting prefix 'test/' from 'test/b.test' resulting in 'b.test' for comparison",
				"Omitting prefix 'test/' from 'test/c.test' resulting in 'c.test' for comparison",
				"Omitting prefix 'test/' from 'test/d.test' resulting in 'd.test' for comparison",
				"Total Runtime: 10ns",
				"Target Partition Runtime: 5ns",
				"[PART 0 (0.00s)]: Assigned 'test/a.test' (4ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'test/b.test' (3ns) using least runtime strategy",
				"[PART 1 (0.00s)]: Assigned 'test/c.test' (2ns) using least runtime strategy",
				"[PART 0 (0.00s)]: Assigned 'test/d.test' (1ns) using least runtime strategy",
			))
		})

		It("logs the partitioned files for index 0 using client test file paths", func() {
			_ = service.Partition(ctx, cfgWithArgs(0, 2, []string{"tests/*.test"}, " ", false, "test/"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("test/a.test test/d.test"))
		})

		It("logs the partitioned files for index 1 using client test file paths", func() {
			_ = service.Partition(ctx, cfgWithArgs(1, 2, []string{"tests/*.test"}, " ", false, "test/"))
			logMessages := make([]string, 0)
			for _, log := range recordedLogs.FilterLevelExact(zap.InfoLevel).All() {
				logMessages = append(logMessages, log.Message)
			}
			Expect(logMessages).To(ContainElement("test/b.test test/c.test"))
		})
	})
})
