package providers

import "github.com/rwx-research/captain-cli/internal/config"

// TODO: add node_index, node_total
// - make it available to the run command
// - make it accessible with to the generic provider
// - expose it via job_tags

type GenericEnv struct {
	Who            string
	Branch         string
	Sha            string
	CommitMessage  string
	BuildURL       string
	Title          string
	PartitionIndex int `env:"CAPTAIN_PARTITION_INDEX" envDefault:"-1"`
	PartitionTotal int `env:"CAPTAIN_PARTITION_TOTAL" envDefault:"-1"`
}

func (cfg GenericEnv) MakeProvider() Provider {
	return Provider{
		AttemptedBy:   cfg.Who,
		BranchName:    cfg.Branch,
		CommitSha:     cfg.Sha,
		CommitMessage: cfg.CommitMessage,
		ProviderName:  "generic",
		JobTags:       map[string]any{"captain_build_url": cfg.BuildURL},
		Title:         cfg.Title,
		PartitionNodes: config.PartitionNodes{
			Index: cfg.PartitionIndex,
			Total: cfg.PartitionTotal,
		},
	}
}

func MergeGeneric(into GenericEnv, from GenericEnv) GenericEnv {
	into.Who = firstNonempty(from.Who, into.Who)
	into.Branch = firstNonempty(from.Branch, into.Branch)
	into.Sha = firstNonempty(from.Sha, into.Sha)
	return into
}
