// Package cli holds the main business logic in our CLI. This is mainly:
// 1. Triggering the right API calls based on the provided input parameters.
// 2. User-friendly logging
// However, this package _does not_ implement the actual terminal UI. That part is handled by `cmd/captain`.
package cli

import (
	"context"
	"mime"
	"os"
	"path"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/api"
)

// Service is the main CLI service.
type Service struct {
	API api.Client
	Log *zap.SugaredLogger
}

// UploadTestResults is the implementation of `captain upload results`. It takes a number of filepaths, checks that
// files exist and uploads them using the provided API.
func (s Service) UploadTestResults(ctx context.Context, suiteName string, filepaths []string) {
	newArtifacts := make([]api.Artifact, len(filepaths))

	for i, filePath := range filepaths {
		fd, err := os.Open(filePath)
		if err != nil {
			s.Log.Errorf("unable to open file: %s", err)
			return
		}
		defer fd.Close()

		id, err := uuid.NewRandom()
		if err != nil {
			s.Log.Errorf("unable to generate new UUID: %s", err)
			return
		}

		newArtifacts[i] = api.Artifact{
			ExternalID:   id,
			FD:           fd,
			Kind:         api.ArtifactKindTestResult,
			MimeType:     mime.TypeByExtension(path.Ext(fd.Name())),
			Name:         suiteName,
			OriginalPath: filePath,
			Parser:       api.ParserTypeRSpecJSON,
		}
	}

	if err := s.API.UploadArtifacts(ctx, newArtifacts); err != nil {
		s.Log.Errorf("unable to upload test result: %s", err)
	}
}
