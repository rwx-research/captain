package providers

type GenericEnv struct {
	Who    string `env:"CAPTAIN_WHO"`
	Branch string `env:"CAPTAIN_BRANCH"`
	Sha    string `env:"CAPTAIN_SHA"`
}

func MakeGenericProvider(cfg GenericEnv) Provider {
	return Provider{
		AttemptedBy:  cfg.Who,
		BranchName:   cfg.Branch,
		CommitSha:    cfg.Sha,
		ProviderName: "generic",
		JobTags:      makeGenericTags(cfg),
	}
}

func MergeGeneric(into GenericEnv, from GenericEnv) GenericEnv {
	into.Who = firstNonempty(from.Who, into.Who)
	into.Branch = firstNonempty(from.Branch, into.Branch)
	into.Sha = firstNonempty(from.Sha, into.Sha)
	return into
}

func makeGenericTags(cfg GenericEnv) map[string]any {
	return map[string]any{}
}
