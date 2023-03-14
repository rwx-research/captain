package remote

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
func (cfg ClientConfig) Validate() error {
	if cfg.Log == nil {
		return errors.NewInternalError("missing logger")
	}

	if cfg.Token == "" {
		return errors.NewConfigurationError("missing API token")
	}

	providerErr := cfg.Provider.Validate()
	if providerErr != nil {
		return errors.Wrap(providerErr, "invalid provider")
	}

	return nil
}

// WithDefaults returns a copy of the configuration with defaults applied where necessary.
func (cfg ClientConfig) WithDefaults() ClientConfig {
	if cfg.Host == "" {
		cfg.Host = defaultHost
	}

	return cfg
}
