// Package testing holds any domain logic and domain models related to testing.
package testing

import (
	"fmt"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// TestResult is a generic representation of a test result
type TestResult struct {
	Description   string
	Duration      time.Duration
	Status        TestStatus
	StatusMessage string
	Meta          map[string]any
}

// Calculates the composite identifier of a TestResult given the components which determine it
func (testResult TestResult) Identify(withComponents []string, strictly bool) (string, error) {
	foundComponents := make([]string, 0)

	for _, component := range withComponents {
		if component == "description" {
			foundComponents = append(foundComponents, testResult.Description)
			continue
		}

		if strictly && testResult.Meta == nil {
			return "", errors.NewInternalError(
				"Meta is not defined for %v, but we tried to get %s from it.",
				testResult,
				component,
			)
		}

		value, exists := testResult.Meta[component]
		if strictly && !exists {
			return "", errors.NewInternalError(
				"Tried to get %s from %v of %v, but it was not there.",
				component,
				testResult.Meta,
				testResult,
			)
		}

		switch {
		case exists && value == nil:
			foundComponents = append(foundComponents, "")
		case exists:
			foundComponents = append(foundComponents, fmt.Sprintf("%v", value))
		default:
			foundComponents = append(foundComponents, "MISSING_IDENTITY_COMPONENT")
		}
	}

	return strings.Join(foundComponents, " -captain- "), nil
}
