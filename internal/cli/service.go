// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/backend"
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
