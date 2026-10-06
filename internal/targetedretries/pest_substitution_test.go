package targetedretries_test

import (
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PestSubstitution", func() {
	DescribeTable("validates the file and filter placeholders", func(template string, valid bool) {
		compiled, err := templating.CompileTemplate(template)
		Expect(err).NotTo(HaveOccurred())
		err = (targetedretries.PestSubstitution{}).ValidateTemplate(compiled)
		if valid {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(ContainSubstring("Retrying Pest")))
		}
	},
		Entry("example", (targetedretries.PestSubstitution{}).Example(), true),
		Entry("reversed", "pest '{{ file }}' --filter '{{ filter }}'", true),
		Entry("no placeholders", "pest", false),
		Entry("missing filter", "pest '{{ file }}'", false),
		Entry("wrong placeholder", "pest '{{ file }}' '{{ name }}'", false),
		Entry("extra placeholder", "pest '{{ file }}' '{{ filter }}' '{{ name }}'", false),
	)

	It("escapes, anchors, groups and deduplicates only selected test names", func() {
		failed := v1.Test{
			Location: &v1.Location{File: "tests/it's.php"},
			Attempt: v1.TestAttempt{
				Meta:   map[string]any{"class": `Tests\Example`, "name": "it's / [x]+$"},
				Status: v1.NewFailedTestStatus(nil, nil, nil),
			},
		}
		passed := failed
		passed.Attempt.Status = v1.NewSuccessfulTestStatus()
		passed.Attempt.Meta = map[string]any{"class": `Tests\Example`, "name": "unrelated"}
		substitution := targetedretries.SubstitutionsByFramework[v1.PHPPestFramework]
		compiled, err := templating.CompileTemplate(substitution.Example())
		Expect(err).NotTo(HaveOccurred())
		values, err := substitution.SubstitutionsFor(compiled, v1.TestResults{Tests: []v1.Test{
			failed, passed, failed,
		}}, func(test v1.Test) bool { return test.Attempt.Status.ImpliesFailure() })
		Expect(err).NotTo(HaveOccurred())
		Expect(values).To(Equal([]map[string]string{{
			"file":   `tests/it'"'"'s.php`,
			"filter": `/\A(?:Tests\\Example::it'"'"'s \/ \[x\]\+\$)\z/u`,
		}}))
	})

	It("rejects native PHPUnit logical locations rather than retrying an invented path", func() {
		values, err := (targetedretries.PestSubstitution{}).SubstitutionsFor(
			templating.CompiledTemplate{}, v1.TestResults{Tests: []v1.Test{{
				Name: "NativeTest::Native method", Location: &v1.Location{File: "Native::Native method"},
			}}}, func(v1.Test) bool { return true },
		)
		Expect(values).To(BeNil())
		Expect(err).To(MatchError(ContainSubstring("separate PHPUnit suite")))
	})
})
