package v1

import "sort"

// Merges test results of the same framework together, indifferent of the original test results groups
func Merge(separateTestResults []TestResults) []TestResults {
	byFramework := make(map[Framework][]TestResults)
	for _, testResults := range separateTestResults {
		byFramework[testResults.Framework] = append(byFramework[testResults.Framework], testResults)
	}

	mergedTestResults := make([]TestResults, 0)
	for _, testResultsToMerge := range byFramework {
		merged, rest := testResultsToMerge[0], testResultsToMerge[1:]
		for _, testResults := range rest {
			merged.DerivedFrom = append(merged.DerivedFrom, testResults.DerivedFrom...)
			merged.OtherErrors = append(merged.OtherErrors, testResults.OtherErrors...)
			merged.Tests = append(merged.Tests, testResults.Tests...)
		}

		merged.Summary = NewSummary(merged.Tests, merged.OtherErrors)
		mergedTestResults = append(mergedTestResults, merged)
	}

	// Primarily for determinism, we don't strictly need this to be sorted any particular way
	sort.Slice(mergedTestResults, func(i, j int) bool {
		if mergedTestResults[i].Framework.Language == mergedTestResults[j].Framework.Language {
			return mergedTestResults[i].Framework.Kind < mergedTestResults[j].Framework.Kind
		}

		return mergedTestResults[i].Framework.Language < mergedTestResults[j].Framework.Language
	})

	return mergedTestResults
}
