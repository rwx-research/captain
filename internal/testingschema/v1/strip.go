package v1

import (
	"encoding/base64"
	"encoding/json"
)

const TruncationMessage = "<truncated due to test results size>"

func StripDerivedFrom(testResults TestResults) TestResults {
	cleanedDerivedFrom := make([]OriginalTestResults, len(testResults.DerivedFrom))

	for i, originalTestResults := range testResults.DerivedFrom {
		cleanedDerivedFrom[i] = OriginalTestResults{
			OriginalFilePath: originalTestResults.OriginalFilePath,
			GroupNumber:      originalTestResults.GroupNumber,
			Contents:         base64.StdEncoding.EncodeToString([]byte(TruncationMessage)),
		}
	}

	return TestResults{
		Framework:   testResults.Framework,
		Summary:     testResults.Summary,
		Tests:       testResults.Tests,
		OtherErrors: testResults.OtherErrors,
		DerivedFrom: cleanedDerivedFrom,
		Meta:        testResults.Meta,
	}
}

func StripPreviousAttempts(testResults TestResults) TestResults {
	for i, test := range testResults.Tests {
		for j, attempt := range test.PastAttempts {
			testResults.Tests[i].PastAttempts[j].Status = stripStatus(attempt.Status)
		}
	}

	return testResults
}

func StripCurrentAttempts(testResults TestResults) TestResults {
	for i, test := range testResults.Tests {
		if test.Attempt.Status.Backtrace != nil {
			testResults.Tests[i].Attempt.Status = stripStatus(test.Attempt.Status)
		}
	}

	return testResults
}

func stripStatus(status TestStatus) TestStatus {
	if status.Backtrace != nil {
		status.Backtrace = []string{TruncationMessage}
	}

	if status.OriginalStatus != nil {
		s := stripStatus(*status.OriginalStatus)
		status.OriginalStatus = &s
	}

	return status
}

func StripToSize(testResults TestResults, fileSizeThresholdBytes int64) TestResults {
	type stripFunc func(TestResults) TestResults
	for _, strip := range []stripFunc{StripDerivedFrom, StripPreviousAttempts, StripCurrentAttempts} {
		// Marshal to check current size
		buf, err := json.Marshal(testResults)
		if err != nil {
			return testResults
		}

		if int64(len(buf)) <= fileSizeThresholdBytes {
			break
		}

		testResults = strip(testResults)
	}

	return testResults
}
