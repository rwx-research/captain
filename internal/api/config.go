package api

import (
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
)

// ClientConfig is the configuration object for the Captain API client
type ClientConfig struct {
	AccountName    string
	Host           string
	Insecure       bool
	Log            *zap.SugaredLogger
	JobName        string
	JobMatrix      *string
	RepositoryName string
	RunAttempt     string
	RunID          string
	Token          string
}

// Validate checks the configuration for errors
func (cc ClientConfig) Validate() error {
	if cc.AccountName == "" {
		return errors.NewConfigurationError("missing account name")
	}

	if cc.Log == nil {
		return errors.NewInternalError("missing logger")
	}

	if cc.JobName == "" {
		return errors.NewConfigurationError("missing job name")
	}

	if cc.RunAttempt == "" {
		return errors.NewConfigurationError("missing run attempt")
	}

	if cc.RunID == "" {
		return errors.NewConfigurationError("missing run ID")
	}

	if cc.RepositoryName == "" {
		return errors.NewConfigurationError("missing repository name")
	}

	if cc.Token == "" {
		return errors.NewConfigurationError("missing API token")
	}

	return nil
}

// WithDefaults returns a copy of the configuration with defaults applied where necessary.
func (cc ClientConfig) WithDefaults() ClientConfig {
	if cc.Host == "" {
		cc.Host = defaultHost
	}

	return cc
}
