package remote

func uniqueStrings(in []string) []string {
	set := make(map[string]struct{})

	for _, s := range in {
		set[s] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}

	return out
}
