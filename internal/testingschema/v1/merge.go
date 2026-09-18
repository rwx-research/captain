package v1

// Merges and flattens test results together
func Merge(allTestResults ...[]TestResults) TestResults {
	unionedTestResults := make([]TestResults, 0)
	for _, batch := range allTestResults {
		if results := union(batch); results != nil {
			unionedTestResults = append(unionedTestResults, *results)
		}
	}

	return flatten(unionedTestResults)
}

func union(separateTestResults []TestResults) *TestResults {
	if len(separateTestResults) == 0 {
		return nil
	}

	unioned, rest := separateTestResults[0], separateTestResults[1:]
	for _, testResults := range rest {
		unioned.DerivedFrom = append(unioned.DerivedFrom, testResults.DerivedFrom...)
		unioned.OtherErrors = append(unioned.OtherErrors, testResults.OtherErrors...)
		unioned.Tests = append(unioned.Tests, testResults.Tests...)
	}

	unioned.Summary = NewSummary(unioned.Tests, unioned.OtherErrors)
	return &unioned
}

func flatten(unionedTestResults []TestResults) TestResults {
	flattened, rest := unionedTestResults[0], unionedTestResults[1:]
	flattenedStartedEmpty := len(flattened.Tests) == 0 &&
		len(flattened.OtherErrors) == 0 &&
		len(flattened.DerivedFrom) == 0

	for index, testResults := range rest {
		flattened.DerivedFrom = append(flattened.DerivedFrom, testResults.DerivedFrom...)
		flattened.OtherErrors = append(flattened.OtherErrors, testResults.OtherErrors...)

		matching := MatchRetries(flattened, []TestResults{testResults})
		for incomingIndex, incomingTest := range testResults.Tests {
			i := matching.OriginalIndex[incomingIndex]
			if i >= 0 {
				baseTest := flattened.Tests[i]

				newAttempt := incomingTest.Attempt
				newPastAttempt := baseTest.Attempt
				if newAttempt.Status.ImpliesSkipped() {
					// do not flatten skipped statuses into existing tests because they didn't actually run again
					continue
				}
				swapped := false
				if newAttempt.Status.ImpliesFailure() && !newPastAttempt.Status.ImpliesFailure() {
					newAttempt, newPastAttempt = newPastAttempt, newAttempt
					swapped = true
				}

				// Preserve the complete attempt history from both sides. The incoming test may carry its
				// own PastAttempts (e.g. a framework's in-process retries within a single Captain
				// invocation); dropping them loses attempts and their distinct attachments (traces).
				pastAttempts := make([]TestAttempt, 0, len(baseTest.PastAttempts)+len(incomingTest.PastAttempts)+1)
				pastAttempts = append(pastAttempts, baseTest.PastAttempts...)
				if swapped {
					// headline stayed on baseTest.Attempt; incomingTest.Attempt becomes the latest past attempt
					pastAttempts = append(pastAttempts, incomingTest.PastAttempts...)
					pastAttempts = append(pastAttempts, newPastAttempt)
				} else {
					// headline moved to incomingTest.Attempt; baseTest.Attempt precedes incoming's past attempts
					pastAttempts = append(pastAttempts, newPastAttempt)
					pastAttempts = append(pastAttempts, incomingTest.PastAttempts...)
				}

				flattened.Tests[i] = Test{
					Scope:        baseTest.Scope,
					ID:           baseTest.ID,
					Name:         baseTest.Name,
					Lineage:      baseTest.Lineage,
					Location:     mergedLocation(baseTest.Location, incomingTest.Location),
					Attempt:      newAttempt,
					PastAttempts: pastAttempts,
				}
			} else {
				if flattenedStartedEmpty && index == 0 {
					flattened.Tests = append(flattened.Tests, incomingTest)
				} else {
					flattened.Tests = append(flattened.Tests, incomingTest.Tag("missingInPreviousBatchOfResults", true))
				}
			}
		}
	}

	flattened.Summary = NewSummary(flattened.Tests, flattened.OtherErrors)
	return flattened
}
