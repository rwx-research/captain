package v1

// RetryMatching describes a one-to-one reconciliation against an immutable batch.
type RetryMatching struct {
	// OriginalIndex is indexed by retry; -1 means no unique match.
	OriginalIndex []int
	Matched       []bool
	Ambiguous     []bool
}

// MatchRetries uses the same reconciliation for merging and retry diagnostics.
// Results in retries belong to one invocation batch, not successive attempts.
func MatchRetries(original TestResults, retries []TestResults) RetryMatching {
	matching := RetryMatching{
		Matched:   make([]bool, len(original.Tests)),
		Ambiguous: make([]bool, len(original.Tests)),
	}
	batch := union(retries)
	if batch == nil {
		return matching
	}
	jest := original.Framework.Kind == FrameworkKindJest && batch.Framework.Kind == FrameworkKindJest

	buckets := make(map[string][]int, len(original.Tests))
	for i, test := range original.Tests {
		key := retryIdentity(test, jest)
		buckets[key] = append(buckets[key], i)
	}

	candidates := make([][]int, len(batch.Tests))
	originalCounts := make([]int, len(original.Tests))
	for retryIndex, retry := range batch.Tests {
		for _, i := range buckets[retryIdentity(retry, jest)] {
			if jest && !compatiblePosition(original.Tests[i].Location, retry.Location) {
				continue
			}
			candidates[retryIndex] = append(candidates[retryIndex], i)
			originalCounts[i]++
		}
	}

	for _, compatible := range candidates {
		index := -1
		if len(compatible) == 1 && originalCounts[compatible[0]] == 1 {
			index = compatible[0]
			matching.Matched[index] = true
		} else {
			for _, i := range compatible {
				matching.Ambiguous[i] = true
			}
		}
		matching.OriginalIndex = append(matching.OriginalIndex, index)
	}
	return matching
}

func retryIdentity(test Test, ignorePosition bool) string {
	if ignorePosition && test.Location != nil {
		location := *test.Location
		location.Line, location.Column = nil, nil
		test.Location = &location
	}
	return test.IdentityForMatching()
}

func compatiblePosition(original, retry *Location) bool {
	if original == nil || retry == nil {
		return original == nil && retry == nil
	}
	return (original.Line == nil || retry.Line == nil || *original.Line == *retry.Line) &&
		(original.Column == nil || retry.Column == nil || *original.Column == *retry.Column)
}

func mergedLocation(original, retry *Location) *Location {
	if original == nil || retry == nil {
		return original
	}
	location := *original
	if location.Line == nil {
		location.Line = retry.Line
	}
	if location.Column == nil {
		location.Column = retry.Column
	}
	return &location
}
