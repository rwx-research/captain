package cli

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	testFilePaths, err := s.FileSystem.GlobMany(cfg.TestFilePaths)
	if err != nil {
		return PartitionResult{}, errors.NewSystemError("unable to expand filepath glob: %s", err)
	}

	if cfg.RoundRobin {
		unmatchedFilepaths = append(unmatchedFilepaths, testFilePaths...)
	} else {
		fileTimings, err := s.API.GetTestTimingManifest(ctx, cfg.SuiteID)
		if err != nil {
			return PartitionResult{}, errors.WithStack(err)
		}

		// Compare expanded client file paths w/ expanded server file paths
		// taking care to always use the client path and sort by duration desc
		for _, clientTestFile := range testFilePaths {
			match := false
			var fileTimingMatch testing.FileTimingMatch
			clientExpandedFilepath, err := filepath.Abs(clientTestFile)
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
	var totalCapacity time.Duration
	for _, fileTimingMatch := range fileTimingMatches {
		totalCapacity += fileTimingMatch.Duration()
	}
	partitionCapacity := totalCapacity / time.Duration(cfg.PartitionNodes.Total)

	s.Log.Debugf("Total Capacity: %s", totalCapacity)
	s.Log.Debugf("Target Partition Capacity: %s", partitionCapacity)

	for i := 0; i < cfg.PartitionNodes.Total; i++ {
		partitions = append(partitions, testing.TestPartition{
			Index:             i,
			TestFilePaths:     make([]string, 0),
			RemainingCapacity: partitionCapacity,
			TotalCapacity:     partitionCapacity,
		})
	}

	for _, fileTimingMatch := range fileTimingMatches {
		fits, partition := partitionWithFirstFit(partitions, fileTimingMatch)
		if fits {
			partition = partition.Add(fileTimingMatch)
			partitions[partition.Index] = partition
			s.Log.Debugf("%s: Assigned %s using first fit strategy", partition, fileTimingMatch)
			continue
		}
		partition = partitionWithMostRemainingCapacity(partitions)
		partition = partition.Add(fileTimingMatch)
		partitions[partition.Index] = partition
		s.Log.Debugf("%s: Assigned %s using most remaining capacity strategy", partition, fileTimingMatch)
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

func partitionWithFirstFit(
	partitions []testing.TestPartition,
	fileTimingMatch testing.FileTimingMatch,
) (fit bool, result testing.TestPartition) {
	for _, p := range partitions {
		if p.RemainingCapacity >= fileTimingMatch.Duration() {
			return true, p
		}
	}
	return false, result
}

func partitionWithMostRemainingCapacity(partitions []testing.TestPartition) testing.TestPartition {
	result := partitions[0]
	for i := 1; i < len(partitions); i++ {
		p := partitions[i]
		if p.RemainingCapacity > result.RemainingCapacity {
			result = p
		}
	}
	return result
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
