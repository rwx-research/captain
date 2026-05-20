package targetedretries_test

import (
	"os"
	"sort"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptTestCafeSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptTestCafeSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.JavaScriptTestCafeSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/testcafe.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptTestCafeParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["file"] != substitutions[j]["file"] {
				return substitutions[i]["file"] < substitutions[j]["file"]
			}

			return substitutions[i]["grep"] < substitutions[j]["grep"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx testcafe chrome")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ wat }}' -T '{{ foo }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the file and grep placeholders", func() {
			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns tests grouped by file using the test name from lineage", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file1 := "tests/checkout.testcafe.ts"
			file2 := "tests/with space.testcafe.ts"

			fixtureName := "Checkout flow"
			testName1 := "applies coupon 'A' + 1"
			testName2 := "shows tax breakdown"
			testName3 := "honors loyalty discount"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Name:     fixtureName + " " + testName1,
						Lineage:  []string{fixtureName, testName1},
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Name:     fixtureName + " " + testName1,
						Lineage:  []string{fixtureName, testName1},
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Name:     fixtureName + " " + testName2,
						Lineage:  []string{fixtureName, testName2},
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewTimedOutTestStatus(nil, nil, nil)},
					},
					{
						Name:     fixtureName + " " + testName3,
						Lineage:  []string{fixtureName, testName3},
						Location: &v1.Location{File: file1},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
					{
						Name:     fixtureName + " " + testName3,
						Lineage:  []string{fixtureName, testName3},
						Location: &v1.Location{File: file2},
						Attempt:  v1.TestAttempt{Status: v1.NewSkippedTestStatus(nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)
			Expect(err).NotTo(HaveOccurred())
			sort.SliceStable(substitutions, func(i int, j int) bool {
				return substitutions[i]["file"] < substitutions[j]["file"]
			})
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file": "tests/checkout.testcafe.ts",
						"grep": `^applies coupon '"'"'A'"'"' \+ 1|shows tax breakdown$`,
					},
					{
						"file": `tests/with space.testcafe.ts`,
						"grep": `^applies coupon '"'"'A'"'"' \+ 1$`,
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file := "tests/checkout.testcafe.ts"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Lineage:  []string{"Checkout flow", "applies coupon code"},
						Location: &v1.Location{File: file},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  []string{"Checkout flow", "shows tax breakdown"},
						Location: &v1.Location{File: file},
						Attempt:  v1.TestAttempt{Status: v1.NewCanceledTestStatus()},
					},
					{
						Lineage:  []string{"Checkout flow", "honors loyalty discount"},
						Location: &v1.Location{File: file},
						Attempt:  v1.TestAttempt{Status: v1.NewSuccessfulTestStatus()},
					},
				},
			}

			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file": "tests/checkout.testcafe.ts",
						"grep": `^applies coupon code$`,
					},
				},
			))
		})

		It("deduplicates repeated test names within the same file", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			file := "tests/checkout.testcafe.ts"

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Lineage:  []string{"Checkout flow", "applies coupon code"},
						Location: &v1.Location{File: file},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  []string{"Checkout flow", "applies coupon code"},
						Location: &v1.Location{File: file},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"file": "tests/checkout.testcafe.ts",
						"grep": `^applies coupon code$`,
					},
				},
			))
		})

		It("skips tests with no location or empty file", func() {
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx testcafe chrome '{{ file }}' -T '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Lineage:  []string{"Fixture", "test with nil location"},
						Location: nil,
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
					{
						Lineage:  []string{"Fixture", "test with empty file"},
						Location: &v1.Location{File: ""},
						Attempt:  v1.TestAttempt{Status: v1.NewFailedTestStatus(nil, nil, nil)},
					},
				},
			}

			substitution := targetedretries.JavaScriptTestCafeSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(BeEmpty())
		})
	})
})
