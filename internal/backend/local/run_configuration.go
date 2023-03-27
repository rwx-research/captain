package local

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/backend"
)

func newCompositeID(identity Map) string {
	components := make([]string, len(identity.Order))

	for i, key := range identity.Order {
		components[i] = identity.Values[key]
	}

	return strings.Join(components, " -captain- ")
}

func makeRunConfiguration(flakes, quarantines []yaml.Node, modTime time.Time) (backend.RunConfiguration, error) {
	config := backend.RunConfiguration{
		GeneratedAt:      time.Now().Format(time.RFC3339),
		QuarantinedTests: make([]backend.QuarantinedTest, len(quarantines)),
		FlakyTests:       make([]backend.Test, len(flakes)),
	}

	for i, flake := range flakes {
		identity, strict := NewMapFromYAML(flake).withoutKey("strict")

		config.FlakyTests[i] = backend.Test{
			CompositeIdentifier: newCompositeID(identity),
			IdentityComponents:  identity.Order,
			StrictIdentity:      strict == "true",
		}
	}

	for i, quarantine := range quarantines {
		identity, strict := NewMapFromYAML(quarantine).withoutKey("strict")

		config.QuarantinedTests[i] = backend.QuarantinedTest{
			Test: backend.Test{
				CompositeIdentifier: newCompositeID(identity),
				IdentityComponents:  identity.Order,
				StrictIdentity:      strict == "true",
			},
			QuarantinedAt: modTime.Format(time.RFC3339),
		}
	}

	return config, nil
}
