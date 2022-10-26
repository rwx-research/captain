package cli

// RunConfig holds the configuration for running a test suite (used by `RunSuite`)
type RunConfig struct {
	Args              []string
	ArtifactGlob      string
	FailOnUploadError bool
	SuiteName         string
}
