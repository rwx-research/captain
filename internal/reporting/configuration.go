package reporting

import "github.com/rwx-research/captain-cli/internal/providers"

type Configuration struct {
	CloudEnabled          bool
	CloudHost             string
	CloudOrganizationSlug string
	SuiteID               string
	RetryCommandTemplate  string
	Provider              providers.Provider
}
