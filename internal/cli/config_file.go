package cli

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// RetriesDisabledValue marks a suite as intentionally having no retries, as `--retries disabled` or `retries: disabled`.
const RetriesDisabledValue = "disabled"

// configFile holds all options that can be set over the config file
type ConfigFile struct {
	Cloud struct {
		APIHost  string `yaml:"api-host" env:"CAPTAIN_HOST"`
		Disabled bool
		Insecure bool
	}
	Flags  map[string]any
	Output struct {
		Debug bool
	}
	TestSuites map[string]SuiteConfig `yaml:"test-suites"`
}

type SuiteConfigOutput struct {
	PrintSummary bool `yaml:"print-summary"`
	Reporters    map[string]string
	Quiet        bool
}

type SuiteConfigResults struct {
	Framework string
	Language  string
	Path      string
}

type SuiteConfigRetries struct {
	Attempts                  int
	Command                   string
	FailFast                  bool     `yaml:"fail-fast"`
	FailOnMisconfiguration    bool     `yaml:"fail-on-misconfiguration"`
	FlakyAttempts             int      `yaml:"flaky-attempts"`
	MaxTests                  string   `yaml:"max-tests"`
	MaxTestsLegacyName        string   `yaml:"maxtests"`
	PostRetryCommands         []string `yaml:"post-retry-commands"`
	PreRetryCommands          []string `yaml:"pre-retry-commands"`
	IntermediateArtifactsPath string   `yaml:"intermediate-artifacts-path"`
	AdditionalArtifactPaths   []string `yaml:"additional-artifact-paths"`
	QuarantinedAttempts       int      `yaml:"quarantined-attempts"`
	Disabled                  bool     `yaml:"-"`
}

// UnmarshalYAML accepts `retries: disabled` in addition to the usual mapping. It uses the callback form so that the
// config file decoder keeps rejecting unknown fields inside the mapping.
func (r *SuiteConfigRetries) UnmarshalYAML(unmarshal func(any) error) error {
	var value string
	if err := unmarshal(&value); err == nil {
		if value != RetriesDisabledValue {
			return &yaml.TypeError{Errors: []string{
				fmt.Sprintf("retries: expected %q or a mapping, got %q", RetriesDisabledValue, value),
			}}
		}

		*r = SuiteConfigRetries{Disabled: true}
		return nil
	}

	type plain SuiteConfigRetries
	return unmarshal((*plain)(r))
}

type SuiteConfigPartition struct {
	Command            string
	Globs              []string
	DiscoveryCommand   string `yaml:"discovery-command"`
	DiscoveryDelimiter string `yaml:"discovery-delimiter"`
	Delimiter          string
	RoundRobin         bool   `yaml:"round-robin"`
	TrimPrefix         string `yaml:"trim-prefix"`
}

// SuiteConfig holds options that can be customized per suite
type SuiteConfig struct {
	Command               string
	FailOnUploadError     bool `yaml:"fail-on-upload-error"`
	FailOnDuplicateTestID bool `yaml:"fail-on-duplicate-test-id"`
	Output                SuiteConfigOutput
	Results               SuiteConfigResults
	Retries               SuiteConfigRetries
	Partition             SuiteConfigPartition
}
