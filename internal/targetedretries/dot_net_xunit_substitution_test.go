package targetedretries_test

import (
	"os"
	"sort"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DotNetxUnitSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.DotNetxUnitSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with a real file", func() {
		substitution := targetedretries.DotNetxUnitSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/xunit_dot_net.xml")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.DotNetxUnitParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(_ v1.Test) bool { return true },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			return substitutions[i]["filter"] < substitutions[j]["filter"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.DotNetxUnitSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.DotNetxUnitSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too many placeholders", func() {
			substitution := targetedretries.DotNetxUnitSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ filter }}' {{ other }}")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template without a filter placeholder", func() {
			substitution := targetedretries.DotNetxUnitSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ other }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with only a filter placeholder", func() {
			substitution := targetedretries.DotNetxUnitSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ filter }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		It("returns the unique test type.method", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ filter }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			type1 := "type1"
			method1 := "method1"
			type2 := "type2"
			method2 := "method2"
			type3 := "type3"
			method3 := "method3"
			type4 := "type4"
			method4 := "method4"
			type5 := "type5"
			method5 := "method5"
			type6 := "type6"
			method6 := "method6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type1, "method": &method1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type2, "method": &method2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type3, "method": &method3},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type3, "method": &method3},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type4, "method": &method4},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type5, "method": &method5},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type6, "method": &method6},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.DotNetxUnitSubstitution{}
			Expect(substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(_ v1.Test) bool { return true },
			)).To(Equal(
				[]map[string]string{
					{
						"filter": "FullyQualifiedName=type1.method1 | " +
							"FullyQualifiedName=type2.method2 | FullyQualifiedName=type3.method3",
					},
				},
			))
		})

		It("filters the tests with the provided function", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ filter }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			type1 := "type1"
			method1 := "method1"
			type2 := "type2"
			method2 := "method2"
			type3 := "type3"
			method3 := "method3"
			type4 := "type4"
			method4 := "method4"
			type5 := "type5"
			method5 := "method5"
			type6 := "type6"
			method6 := "method6"
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type1, "method": &method1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type2, "method": &method2},
							Status: v1.NewCanceledTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type3, "method": &method3},
							Status: v1.NewTimedOutTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type3, "method": &method3},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type4, "method": &method4},
							Status: v1.NewPendedTestStatus(nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type5, "method": &method5},
							Status: v1.NewSuccessfulTestStatus(),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type6, "method": &method6},
							Status: v1.NewSkippedTestStatus(nil),
						},
					},
				},
			}

			substitution := targetedretries.DotNetxUnitSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"filter": "FullyQualifiedName=type1.method1 | FullyQualifiedName=type3.method3",
					},
				},
			))
		})

		It("correctly escapes the filter substitution", func() {
			compiledTemplate, compileErr := templating.CompileTemplate("dotnet test --filter '{{ filter }}'")
			Expect(compileErr).NotTo(HaveOccurred())

			type1 := "type1"
			method1 := `method1(val1: 100, val2: "test")`
			type2 := "type2"
			method2 := `!method2=|&\`
			testResults := v1.TestResults{
				Tests: []v1.Test{
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type1, "method": &method1},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
					{
						Attempt: v1.TestAttempt{
							Meta:   map[string]any{"type": &type2, "method": &method2},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					},
				},
			}

			substitution := targetedretries.DotNetxUnitSubstitution{}
			substitutions, err := substitution.SubstitutionsFor(
				compiledTemplate,
				testResults,
				func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(substitutions).To(Equal(
				[]map[string]string{
					{
						"filter": `FullyQualifiedName=type1.method1\(val1: 100, val2: "test"\) | ` +
							`FullyQualifiedName=type2.\!method2\=\|\&\\`,
					},
				},
			))
		})
	})
})
