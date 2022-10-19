package cli

import (
	"context"

	"github.com/rwx-research/captain-cli/internal/api"
)

type APIClient interface {
	GetQuarantinedTestIDs(context.Context) ([]string, error)
	UploadArtifacts(context.Context, []api.Artifact) error
}
