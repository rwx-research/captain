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

		for _, incomingTest := range testResults.Tests {
			matchedWithBaseTest := false

			for i, baseTest := range flattened.Tests {
				if !baseTest.Matches(incomingTest) {
					continue
				}
				matchedWithBaseTest = true

				newAttempt := incomingTest.Attempt
				newPastAttempt := baseTest.Attempt
				if newAttempt.Status.ImpliesSkipped() {
					// do not flatten skipped statuses into existing tests because they didn't actually run again
					break
				}
				if newAttempt.Status.ImpliesFailure() && !newPastAttempt.Status.ImpliesFailure() {
					newAttempt, newPastAttempt = newPastAttempt, newAttempt
				}

				pastAttempts := make([]TestAttempt, len(baseTest.PastAttempts)+1)
				copy(pastAttempts, baseTest.PastAttempts)
				pastAttempts[len(pastAttempts)-1] = newPastAttempt

				flattened.Tests[i] = Test{
					Scope:        baseTest.Scope,
					ID:           baseTest.ID,
					Name:         baseTest.Name,
					Lineage:      baseTest.Lineage,
					Location:     baseTest.Location,
					Attempt:      newAttempt,
					PastAttempts: pastAttempts,
				}
				break
			}

			if !matchedWithBaseTest {
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
