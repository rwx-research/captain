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

	index := NewTestIndex(flattened.Tests)

	for batch, testResults := range rest {
		flattened.DerivedFrom = append(flattened.DerivedFrom, testResults.DerivedFrom...)
		flattened.OtherErrors = append(flattened.OtherErrors, testResults.OtherErrors...)

		for _, incomingTest := range testResults.Tests {
			i := index.IndexOfTestToFlattenInto(flattened.Tests, incomingTest)

			if i < 0 {
				if flattenedStartedEmpty && batch == 0 {
					flattened.Tests = append(flattened.Tests, incomingTest)
				} else {
					incomingTest = incomingTest.Tag("missingInPreviousBatchOfResults", true)
					flattened.Tests = append(flattened.Tests, incomingTest)
				}
				index.add(len(flattened.Tests)-1, incomingTest)
				continue
			}

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
				Location:     mostSpecificLocation(baseTest.Location, incomingTest.Location),
				Attempt:      newAttempt,
				PastAttempts: pastAttempts,
			}
		}
	}

	flattened.Summary = NewSummary(flattened.Tests, flattened.OtherErrors)
	return flattened
}

// Groups tests by everything in their identity except the location's line and column, so that a
// test which reports no line on one attempt still has a small set of candidates to match against.
type TestIndex struct {
	candidates map[string][]int
}

func NewTestIndex(tests []Test) TestIndex {
	index := TestIndex{candidates: make(map[string][]int, len(tests))}
	for i, test := range tests {
		index.add(i, test)
	}

	return index
}

func (index TestIndex) add(i int, test Test) {
	key := test.identityForMatching(true, true)
	index.candidates[key] = append(index.candidates[key], i)
}

// Finds the test in tests that incomingTest should be flattened into, or -1 when there isn't one.
func (index TestIndex) IndexOfTestToFlattenInto(tests []Test, incomingTest Test) int {
	incomingIdentity := incomingTest.IdentityForMatching()
	incomingHasPosition := incomingTest.locationHasPosition()
	looseMatch := -1
	looseMatches := 0

	for _, i := range index.candidates[incomingTest.identityForMatching(true, true)] {
		if tests[i].IdentityForMatching() == incomingIdentity {
			return i
		}

		// Test.Matches only reaches further than an identity when one side is missing a line or column.
		if incomingHasPosition && tests[i].locationHasPosition() {
			continue
		}

		if tests[i].Matches(incomingTest) {
			looseMatch = i
			looseMatches++
		}
	}

	// A file can declare the same test name twice, so a loose match can find more than one test.
	if looseMatches == 1 {
		return looseMatch
	}

	return -1
}

// Jest reports no line or column for a test that fails by timeout, but its passing retry reports
// both.
func mostSpecificLocation(baseLocation *Location, incomingLocation *Location) *Location {
	switch {
	case baseLocation == nil:
		return incomingLocation
	case incomingLocation == nil:
		return baseLocation
	case baseLocation.Line == nil && incomingLocation.Line != nil:
		return incomingLocation
	case baseLocation.Column == nil && incomingLocation.Column != nil:
		return incomingLocation
	default:
		return baseLocation
	}
}
