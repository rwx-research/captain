package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mattn/go-shellwords"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/testing"
)

// Partition splits a glob of test filepaths using decreasing first fit backed by a timing manifest from captain.
func (s Service) Partition(ctx context.Context, cfg PartitionConfig) error {
	err := cfg.Validate()
	if err != nil {
		return errors.WithStack(err)
	}
	partitionResult, err := s.calculatePartition(ctx, cfg)
	if err != nil {
		return err
	}
	s.Log.Infoln(strings.Join(partitionResult.partition.TestFilePaths, cfg.Delimiter))
	return nil
}

func (s Service) calculatePartition(ctx context.Context, cfg PartitionConfig) (PartitionResult, error) {
	fileTimingMatches := make([]testing.FileTimingMatch, 0)
	unmatchedFilepaths := make([]string, 0)
	testFilePaths, err := s.discoverPartitionEntities(ctx, cfg)
	if err != nil {
		return PartitionResult{}, err
	}

	if cfg.RoundRobin {
		unmatchedFilepaths = append(unmatchedFilepaths, testFilePaths...)
	} else {
		fileTimings, err := s.API.GetTestTimingManifest(ctx, cfg.SuiteID, cfg.TimingGranularity())
		if err != nil {
			return PartitionResult{}, errors.WithStack(err)
		}

		// Compare expanded client file paths w/ expanded server file paths
		// taking care to always use the client path and sort by duration desc
		for _, clientTestFile := range testFilePaths {
			match := false
			var fileTimingMatch testing.FileTimingMatch
			if cfg.DiscoveryCommand != "" {
				for _, timing := range fileTimings {
					if timing.Filepath == clientTestFile {
						fileTimingMatches = append(fileTimingMatches, testing.FileTimingMatch{
							FileTiming: timing, ClientFilepath: clientTestFile,
						})
						match = true
						break
					}
				}
				if !match {
					unmatchedFilepaths = append(unmatchedFilepaths, clientTestFile)
				}
				continue
			}
			clientExpandedFilepath := clientTestFile
			if cfg.TrimPrefix != "" {
				trimmedClientExpandedFilepath := strings.TrimPrefix(clientTestFile, cfg.TrimPrefix)
				s.Log.Debugf(
					"Trimming prefix '%s' from '%s' resulting in '%s' for comparison",
					cfg.TrimPrefix,
					clientTestFile,
					trimmedClientExpandedFilepath,
				)
				clientExpandedFilepath = trimmedClientExpandedFilepath
			}
			clientExpandedFilepath, err = filepath.Abs(clientExpandedFilepath)
			if err != nil {
				s.Log.Warnf("failed to expand path of test file: %s", clientTestFile)
				unmatchedFilepaths = append(unmatchedFilepaths, clientTestFile)
				continue
			}

			for _, serverTiming := range fileTimings {
				serverExpandedFilepath, err := filepath.Abs(serverTiming.Filepath)
				if err != nil {
					s.Log.Warnf("failed to expand filepath of timing file: %s", serverTiming.Filepath)
					break
				}
				if clientExpandedFilepath == serverExpandedFilepath {
					match = true
					fileTimingMatch = testing.FileTimingMatch{
						FileTiming:     serverTiming,
						ClientFilepath: clientTestFile,
					}
					break
				}
			}
			if match {
				fileTimingMatches = append(fileTimingMatches, fileTimingMatch)
			} else {
				unmatchedFilepaths = append(unmatchedFilepaths, clientTestFile)
			}
		}
		sort.SliceStable(fileTimingMatches, func(i, j int) bool {
			if fileTimingMatches[i].Duration() == fileTimingMatches[j].Duration() {
				return fileTimingMatches[i].ClientFilepath > fileTimingMatches[j].ClientFilepath
			}

			return fileTimingMatches[i].Duration() > fileTimingMatches[j].Duration()
		})

		if len(fileTimingMatches) == 0 {
			s.Log.Warnln("No test file timings were matched. Using naive round-robin strategy.")
		}
	}

	partitions := make([]testing.TestPartition, 0)
	var totalRuntime time.Duration
	for _, fileTimingMatch := range fileTimingMatches {
		totalRuntime += fileTimingMatch.Duration()
	}
	partitionRuntime := totalRuntime / time.Duration(cfg.PartitionNodes.Total)

	s.Log.Debugf("Total Runtime: %s", totalRuntime)
	s.Log.Debugf("Target Partition Runtime: %s", partitionRuntime)

	for i := 0; i < cfg.PartitionNodes.Total; i++ {
		partitions = append(partitions, testing.TestPartition{
			Index:         i,
			TestFilePaths: make([]string, 0),
			Runtime:       time.Duration(0),
		})
	}

	for _, fileTimingMatch := range fileTimingMatches {
		partition := partitionWithLeastRuntime(partitions).Add(fileTimingMatch)
		partitions[partition.Index] = partition
		s.Log.Debugf("%s: Assigned %s using least runtime strategy", partition, fileTimingMatch)
	}

	for i, testFilepath := range unmatchedFilepaths {
		partition := partitions[i%len(partitions)]
		partitions[partition.Index] = partition.AddFilePath(testFilepath)
		s.Log.Debugf("%s: Assigned '%s' using round robin strategy", partition, testFilepath)
	}

	return PartitionResult{
		partition:              partitions[cfg.PartitionNodes.Index],
		utilizedPartitionCount: utilizedPartitionCount(partitions),
	}, nil
}

func (s Service) discoverPartitionEntities(ctx context.Context, cfg PartitionConfig) ([]string, error) {
	if cfg.DiscoveryCommand == "" {
		paths, err := s.FileSystem.GlobMany(cfg.TestFilePaths)
		if err != nil {
			return nil, errors.NewSystemError("unable to expand filepath glob: %s", err)
		}
		return paths, nil
	}
	args, err := shellwords.Parse(cfg.DiscoveryCommand)
	if err != nil || len(args) == 0 {
		return nil, errors.NewInputError("unable to parse discovery-command %q", cfg.DiscoveryCommand)
	}
	var output bytes.Buffer
	if _, err := s.runCommand(ctx, args, &output, nil); err != nil {
		return nil, errors.Wrap(err, "discovery-command failed")
	}
	delimiter := cfg.DiscoveryDelimiter
	if delimiter == "" {
		delimiter = "\n"
	}
	entities := make([]string, 0)
	seen := make(map[string]bool)
	for _, entity := range strings.Split(output.String(), delimiter) {
		if delimiter == "\n" {
			entity = strings.TrimSuffix(entity, "\r")
		}
		if entity != "" && !seen[entity] {
			entities = append(entities, entity)
			seen[entity] = true
		}
	}
	return entities, nil
}

func partitionWithLeastRuntime(partitions []testing.TestPartition) testing.TestPartition {
	selected := partitions[0]

	for _, candidate := range partitions {
		if candidate.Runtime < selected.Runtime {
			selected = candidate
			continue
		}

		if candidate.Runtime == selected.Runtime && len(candidate.TestFilePaths) < len(selected.TestFilePaths) {
			selected = candidate
		}
	}

	return selected
}

func utilizedPartitionCount(partitions []testing.TestPartition) int {
	count := 0
	for _, partition := range partitions {
		if len(partition.TestFilePaths) != 0 {
			count++
		}
	}
	return count
}

type PartitionResult struct {
	partition              testing.TestPartition
	utilizedPartitionCount int
}
