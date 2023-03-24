package local

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rwx-research/captain-cli/internal/backend"
)

// TestConfiguration holds the flaky or quarantined tests for each suite.
// The top level key maps a list of tests to a suite ID: map[<suite-id>][]<test>
// Each test in turn is a map[<id-component>]<value>, for example: map["description"] = "hello world".
// Note that this last map[string]string is represented as a yaml.Node below. This is because we need to preserve the
// order of keys in that map.
type TestConfiguration map[string][]yaml.Node

func filterStrictness(components []string, values map[string]string) ([]string, bool) {
	filtered := make([]string, 0)
	strict := false

	for _, component := range components {
		if component != "strict" {
			filtered = append(filtered, component)
			continue
		}

		if value, ok := values[component]; ok && value == "true" {
			strict = true
		}
	}

	return filtered, strict
}

func makeCompositeIdentifier(identityComponents []string, values map[string]string) string {
	components := make([]string, len(identityComponents))

	for i, key := range identityComponents {
		components[i] = values[key]
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
		identityComponents, identityValues := parseNode(flake)
		identityComponents, strict := filterStrictness(identityComponents, identityValues)

		config.FlakyTests[i] = backend.Test{
			CompositeIdentifier: makeCompositeIdentifier(identityComponents, identityValues),
			IdentityComponents:  identityComponents,
			StrictIdentity:      strict,
		}
	}

	for i, quarantine := range quarantines {
		identityComponents, identityValues := parseNode(quarantine)
		identityComponents, strict := filterStrictness(identityComponents, identityValues)

		config.QuarantinedTests[i] = backend.QuarantinedTest{
			Test: backend.Test{
				CompositeIdentifier: makeCompositeIdentifier(identityComponents, identityValues),
				IdentityComponents:  identityComponents,
				StrictIdentity:      strict,
			},
			QuarantinedAt: modTime.Format(time.RFC3339),
		}
	}

	return config, nil
}

func parseNode(node yaml.Node) ([]string, map[string]string) {
	order := make([]string, 0)
	values := make(map[string]string, 0)

	for i, subNode := range node.Content {
		if i%2 == 0 {
			order = append(order, subNode.Value)
		} else {
			values[order[len(order)-1]] = subNode.Value
		}
	}

	return order, values
}
