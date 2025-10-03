package cli

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
}

type SuiteConfigPartition struct {
	Command    string
	Globs      []string
	Delimiter  string
	RoundRobin bool   `yaml:"round-robin"`
	TrimPrefix string `yaml:"trim-prefix"`
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
