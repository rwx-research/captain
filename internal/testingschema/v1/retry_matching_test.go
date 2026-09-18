package v1_test

import (
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retry reconciliation", func() {
	position := func(line, column int) *v1.Location {
		location := &v1.Location{File: "test.js"}
		if line != 0 {
			location.Line = &line
		}
		if column != 0 {
			location.Column = &column
		}
		return location
	}
	batch := func(framework v1.Framework, locations []*v1.Location, passed bool) v1.TestResults {
		result := v1.TestResults{Framework: framework}
		for _, location := range locations {
			status := v1.NewFailedTestStatus(nil, nil, nil)
			if passed {
				status = v1.NewSuccessfulTestStatus()
			}
			result.Tests = append(result.Tests, v1.Test{
				Name: "same name", Location: location, Attempt: v1.TestAttempt{Status: status},
			})
		}
		return result
	}

	DescribeTable("matches only mutually unique compatible identities",
		func(framework v1.Framework, originalLocations, retryLocations []*v1.Location, expected []int) {
			original := batch(framework, originalLocations, false)
			retry := batch(framework, retryLocations, true)
			matching := v1.MatchRetries(original, []v1.TestResults{retry})
			Expect(matching.OriginalIndex).To(Equal(expected))
			merged := v1.Merge([]v1.TestResults{original}, []v1.TestResults{retry})
			matched := 0
			for _, index := range expected {
				if index >= 0 {
					matched++
					Expect(merged.Tests[index].Attempt.Status.Kind).To(Equal(v1.TestStatusSuccessful))
					Expect(merged.Tests[index].PastAttempts).To(HaveLen(1))
				}
			}
			Expect(merged.Tests).To(HaveLen(len(originalLocations) + len(retryLocations) - matched))
		},
		Entry("missing original position", v1.JavaScriptJestFramework,
			[]*v1.Location{position(0, 0)}, []*v1.Location{position(13, 2)}, []int{0}),
		Entry("missing retry position", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 2)}, []*v1.Location{position(0, 0)}, []int{0}),
		Entry("complementary position components", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 0)}, []*v1.Location{position(0, 2)}, []int{0}),
		Entry("conflicting known line", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 0)}, []*v1.Location{position(14, 2)}, []int{-1}),
		Entry("conflicting known column", v1.JavaScriptJestFramework,
			[]*v1.Location{position(0, 2)}, []*v1.Location{position(13, 3)}, []int{-1}),
		Entry("multiple original candidates", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 2), position(14, 2)}, []*v1.Location{position(0, 0)}, []int{-1}),
		Entry("multiple retry candidates", v1.JavaScriptJestFramework,
			[]*v1.Location{position(0, 0)}, []*v1.Location{position(13, 2), position(14, 2)}, []int{-1, -1}),
		Entry("exact duplicates", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 2), position(13, 2)}, []*v1.Location{position(13, 2)}, []int{-1}),
		Entry("an exact match does not override ambiguity", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 2), position(0, 0)}, []*v1.Location{position(13, 2)}, []int{-1}),
		Entry("distinct positions remain distinguishable", v1.JavaScriptJestFramework,
			[]*v1.Location{position(13, 2), position(14, 2)}, []*v1.Location{position(14, 2), position(13, 2)}, []int{1, 0}),
		Entry("non-Jest remains strict", v1.RubyRSpecFramework,
			[]*v1.Location{position(0, 0)}, []*v1.Location{position(13, 2)}, []int{-1}),
	)

	It("enriches location without mutating the original and still reconciles a failing retry", func() {
		original := batch(v1.JavaScriptJestFramework, []*v1.Location{position(13, 0)}, false)
		originalLocation := original.Tests[0].Location
		retry := batch(v1.JavaScriptJestFramework, []*v1.Location{position(0, 2)}, false)
		Expect(v1.MatchRetries(original, []v1.TestResults{retry}).Matched).To(Equal([]bool{true}))
		merged := v1.Merge([]v1.TestResults{original}, []v1.TestResults{retry})
		Expect(merged.Tests[0].Location).To(Equal(position(13, 2)))
		Expect(originalLocation.Column).To(BeNil())
		Expect(merged.Tests[0].Attempt.Status.Kind).To(Equal(v1.TestStatusFailed))
	})

	It("detects ambiguity across retry result files", func() {
		original := batch(v1.JavaScriptJestFramework, []*v1.Location{position(0, 0)}, false)
		retry := batch(v1.JavaScriptJestFramework, []*v1.Location{position(13, 2)}, true)
		matching := v1.MatchRetries(original, []v1.TestResults{retry, retry})
		Expect(matching.Matched).To(Equal([]bool{false}))
		Expect(matching.Ambiguous).To(Equal([]bool{true}))
		Expect(v1.Merge([]v1.TestResults{original}, []v1.TestResults{retry, retry}).Tests).To(HaveLen(3))
	})
})
