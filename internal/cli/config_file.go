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
	FailFast                  bool `yaml:"fail-fast"`
	FlakyAttempts             int  `yaml:"flaky-attempts"`
	MaxTests                  string
	PostRetryCommands         []string `yaml:"post-retry-commands"`
	PreRetryCommands          []string `yaml:"pre-retry-commands"`
	IntermediateArtifactsPath string   `yaml:"intermediate-artifacts-path"`
}

type SuiteConfigPartition struct {
	Command   string
	Glob      []string
	Delimiter string
}

// SuiteConfig holds options that can be customized per suite
type SuiteConfig struct {
	Command           string
	FailOnUploadError bool `yaml:"fail-on-upload-error"`
	Output            SuiteConfigOutput
	Results           SuiteConfigResults
	Retries           SuiteConfigRetries
	Partition         SuiteConfigPartition
}
