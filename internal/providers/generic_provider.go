package providers

import "github.com/rwx-research/captain-cli/internal/cli"

// TODO: add node_index, node_total
// - make it available to the run command
// - make it accessible with to the generic provider
// - expose it via job_tags

type GenericEnv struct {
	Who            string
	Branch         string
	Sha            string
	CommitMessage  string
	BuildID        string
	JobID          string
	PartitionNodes cli.PartitionNodes
	JobName        string
}

func MakeGenericProvider(cfg GenericEnv) Provider {
	return Provider{
		AttemptedBy:   cfg.Who,
		BranchName:    cfg.Branch,
		CommitSha:     cfg.Sha,
		CommitMessage: cfg.CommitMessage,
		ProviderName:  "generic",
		JobTags:       makeGenericTags(cfg),
	}
}

func MergeGeneric(into GenericEnv, from GenericEnv) GenericEnv {
	into.Who = firstNonempty(from.Who, into.Who)
	into.Branch = firstNonempty(from.Branch, into.Branch)
	into.Sha = firstNonempty(from.Sha, into.Sha)
	return into
}

func makeGenericTags(cfg GenericEnv) map[string]any {
	tags := map[string]any{
		"captain_build_id": cfg.BuildID,
		"captain_job_name": cfg.JobName,
		"captain_job_id":   cfg.JobID,
	}

	if cfg.PartitionNodes.Total >= 2 {
		tags["captain_partition_index"] = cfg.PartitionNodes.Index
		tags["captain_partition_total"] = cfg.PartitionNodes.Total
	}
	return tags
}
