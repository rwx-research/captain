package cli

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args                []string
	TestResultsFileGlob string
	FailOnUploadError   bool
	SuiteID             string
}
