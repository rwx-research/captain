package v1_test

import (
	"fmt"
	"slices"
	"testing"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

func BenchmarkMergeRetries(b *testing.B) {
	for _, scenario := range []struct {
		name            string
		originals       int
		retries         int
		missingPosition bool
		duplicateNames  bool
		retryFirst      bool
	}{
		{name: "unique-100", originals: 100, retries: 100},
		{name: "unique-1000", originals: 1000, retries: 1000},
		{name: "unique-5000", originals: 5000, retries: 5000},
		{name: "sparse-5000-10", originals: 5000, retries: 10},
		{name: "early-5000-10", originals: 5000, retries: 10, retryFirst: true},
		{name: "missing-position-1000", originals: 1000, retries: 1000, missingPosition: true},
		{name: "same-name-distinct-lines-1000", originals: 1000, retries: 1000, duplicateNames: true},
		{name: "ambiguous-1000", originals: 1000, retries: 1000, duplicateNames: true, missingPosition: true},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			original := v1.TestResults{Framework: v1.JavaScriptJestFramework}
			retry := v1.TestResults{Framework: v1.JavaScriptJestFramework}
			for i := 0; i < scenario.originals; i++ {
				name := fmt.Sprintf("test %d", i)
				if scenario.duplicateNames {
					name = "same name"
				}
				line, column := i+1, 1
				location := &v1.Location{File: "suite.test.js", Line: &line, Column: &column}
				if scenario.missingPosition {
					location = &v1.Location{File: "suite.test.js"}
				}
				original.Tests = append(original.Tests, v1.Test{
					Name: name, Lineage: []string{"suite", name}, Location: location,
					Attempt: v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
				})
				if (!scenario.retryFirst && i >= scenario.originals-scenario.retries) ||
					(scenario.retryFirst && i < scenario.retries) {
					retry.Tests = append(retry.Tests, v1.Test{
						Name: name, Lineage: []string{"suite", name},
						Location: &v1.Location{File: "suite.test.js", Line: &line, Column: &column},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					})
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Merge mutates test slices; keep each iteration independent.
				base, incoming := original, retry
				base.Tests, incoming.Tests = slices.Clone(original.Tests), slices.Clone(retry.Tests)
				v1.Merge([]v1.TestResults{base}, []v1.TestResults{incoming})
			}
		})
	}
}
