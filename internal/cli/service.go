// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"context"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend"
	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/parsing"
)

// Service is the main CLI service.
type Service struct {
	API         backend.Client
	Log         *zap.SugaredLogger
	FileSystem  fs.FileSystem
	TaskRunner  TaskRunner
	ParseConfig parsing.Config
}

type contextKey string

var configKey = contextKey("captainService")

func GetService(cmd *cobra.Command) (Service, error) {
	val := cmd.Context().Value(configKey)
	if val == nil {
		return Service{}, errors.NewInternalError(
			"Tried to fetch config from the command but it wasn't set. This should never happen!")
	}

	service, ok := val.(Service)
	if !ok {
		return Service{}, errors.NewInternalError(
			"Tried to fetch config from the command but it was of the wrong type. This should never happen!")
	}

	return service, nil
}

func SetService(cmd *cobra.Command, service Service) error {
	if _, err := GetService(cmd); err == nil {
		return errors.NewInternalError("Tried to set config on the command but it was already set. This should never happen!")
	}

	ctx := context.WithValue(cmd.Context(), configKey, service)
	cmd.SetContext(ctx)
	return nil
}
