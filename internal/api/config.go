package api

import (
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/providers"
)

// ClientConfig is the configuration object for the Captain API client
type ClientConfig struct {
	Debug    bool
	Host     string
	Insecure bool
	Log      *zap.SugaredLogger
	Token    string
	Provider providers.Provider
}

// Validate checks the configuration for errors
func (cc ClientConfig) Validate() error {
	if cc.Log == nil {
		return errors.NewInternalError("missing logger")
	}

	if cc.Token == "" {
		return errors.NewConfigurationError("missing API token")
	}

	providerErr := cc.Provider.Validate()
	if providerErr != nil {
		return errors.Wrap(providerErr, "invalid provider")
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
