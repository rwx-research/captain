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

var _ = Describe("JavaScriptPlaywrightSubstitution", func() {
	It("adheres to the Substitution interface", func() {
		var substitution targetedretries.Substitution = targetedretries.JavaScriptPlaywrightSubstitution{}
		Expect(substitution).NotTo(BeNil())
	})

	It("works with the example template", func() {
		substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/playwright.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["tests"] != substitutions[j]["tests"] {
				return substitutions[i]["tests"] < substitutions[j]["tests"]
			}

			return substitutions[i]["project"] < substitutions[j]["project"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	It("works with the old template format", func() {
		substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(
			"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
		)
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/playwright.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
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

			if substitutions[i]["project"] != substitutions[j]["project"] {
				return substitutions[i]["project"] < substitutions[j]["project"]
			}

			return substitutions[i]["grep"] < substitutions[j]["grep"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	It("works with the new template format", func() {
		substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
		compiledTemplate, compileErr := templating.CompileTemplate(
			"npx playwright test {{ tests }} --project '{{ project }}'",
		)
		Expect(compileErr).NotTo(HaveOccurred())

		err := substitution.ValidateTemplate(compiledTemplate)
		Expect(err).NotTo(HaveOccurred())

		fixture, err := os.Open("../../test/fixtures/playwright.json")
		Expect(err).ToNot(HaveOccurred())

		testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
		Expect(err).ToNot(HaveOccurred())

		substitutions, err := substitution.SubstitutionsFor(
			compiledTemplate,
			*testResults,
			func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
		)
		Expect(err).NotTo(HaveOccurred())
		sort.SliceStable(substitutions, func(i int, j int) bool {
			if substitutions[i]["tests"] != substitutions[j]["tests"] {
				return substitutions[i]["tests"] < substitutions[j]["tests"]
			}

			return substitutions[i]["project"] < substitutions[j]["project"]
		})
		cupaloy.SnapshotT(GinkgoT(), substitutions)
	})

	Describe("Example", func() {
		It("compiles and is valid", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(substitution.Example())
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ValidateTemplate", func() {
		It("is invalid for a template without placeholders", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate("npx playwright test")
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with too few placeholders", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test '{{ file }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with additional placeholders", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}' {{ foo }}",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders for the old format", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test '{{ wat }}' --project '{{ who }}' --grep '{{ foo }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is invalid for a template with incorrect placeholders for the new format", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test {{ wat }} --project '{{ who }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).To(HaveOccurred())
		})

		It("is valid for a template with the file, project, and grep placeholders", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})

		It("is valid for a template with the tests and project placeholders", func() {
			substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
			compiledTemplate, compileErr := templating.CompileTemplate(
				"npx playwright test {{ tests }} --project '{{ project }}'",
			)
			Expect(compileErr).NotTo(HaveOccurred())

			err := substitution.ValidateTemplate(compiledTemplate)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Substitutions", func() {
		Describe("the old template format", func() {
			It("targets the annotated serial suite location without narrowing by test name", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				fixture, err := os.Open("../../test/fixtures/playwright_serial.json")
				Expect(err).ToNot(HaveOccurred())
				testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
				Expect(err).ToNot(HaveOccurred())

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					*testResults,
					func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(substitutions).To(Equal([]map[string]string{
					{
						"file":    "/project/tests/example.spec.ts:10",
						"project": "chromium",
						"grep":    ".*",
					},
				}))
			})

			It("keeps mixed ordinary and deduplicated serial substitutions", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				fixture, err := os.Open("../../test/fixtures/playwright_serial.json")
				Expect(err).ToNot(HaveOccurred())
				testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
				Expect(err).ToNot(HaveOccurred())

				secondSerialFailure := testResults.Tests[0]
				secondSerialFailure.Name = "second serial failure"
				secondSerialFailure.Location = &v1.Location{
					File: "/project/tests/example.spec.ts",
					Line: new(43),
				}
				ordinaryFailure := v1.Test{
					Name:     "ordinary failure",
					Location: &v1.Location{File: "/project/tests/example.spec.ts", Line: new(50)},
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"annotations": []parsing.JavaScriptPlaywrightAnnotation{},
							"project":     "chromium",
						},
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
				}
				wholeFileSerialFailure := v1.Test{
					Name:     "whole file serial failure",
					Location: &v1.Location{File: "/project/tests/example.spec.ts", Line: new(60)},
					Attempt: v1.TestAttempt{
						Meta: map[string]any{
							"annotations": []parsing.JavaScriptPlaywrightAnnotation{
								{
									Type: "rwx:serial",
									Location: &parsing.JavaScriptPlaywrightLocation{
										File: "/project/tests/example.spec.ts",
										Line: 0,
									},
								},
							},
							"project": "chromium",
						},
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
				}
				testResults.Tests = []v1.Test{
					testResults.Tests[0],
					secondSerialFailure,
					ordinaryFailure,
					wholeFileSerialFailure,
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
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
				Expect(substitutions).To(Equal([]map[string]string{
					{
						"file":    "/project/tests/example.spec.ts",
						"project": "chromium",
						"grep":    ".*",
					},
					{
						"file":    "/project/tests/example.spec.ts",
						"project": "chromium",
						"grep":    "ordinary failure",
					},
					{
						"file":    "/project/tests/example.spec.ts:10",
						"project": "chromium",
						"grep":    ".*",
					},
				}))
			})

			It("returns tests grouped by file and project", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				file1 := "path/to/file1.js"
				file2 := "path/to/file with '.js"

				project1 := "project1"
				project2 := "project with ' 2"

				name1 := "name of describe test 'one' + 1"
				name2 := "name of describe test 2"

				testResults := v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     name1,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewCanceledTestStatus(),
							},
						},
						{
							Name:     name1,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewTimedOutTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewPendedTestStatus(nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewSuccessfulTestStatus(),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewSkippedTestStatus(nil),
							},
						},
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
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
							"file":    `path/to/file with '"'"'.js`,
							"project": `project with '"'"' 2`,
							"grep":    `name of describe test '"'"'one'"'"' \+ 1`,
						},
						{
							"file":    "path/to/file1.js",
							"project": "project1",
							"grep":    `name of describe test '"'"'one'"'"' \+ 1|name of describe test 2`,
						},
					},
				))
			})

			It("filters the tests with the provided function", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test '{{ file }}' --project '{{ project }}' --grep '{{ grep }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				file1 := "path/to/file1.js"
				file2 := "path/to/file with '.js"

				project1 := "project1"
				project2 := "project with ' 2"

				name1 := "name of describe test 'one' + 1"
				name2 := "name of describe test 2"

				testResults := v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     name1,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewCanceledTestStatus(),
							},
						},
						{
							Name:     name1,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewTimedOutTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewPendedTestStatus(nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewSuccessfulTestStatus(),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewSkippedTestStatus(nil),
							},
						},
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					testResults,
					func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
				)
				Expect(err).NotTo(HaveOccurred())
				sort.SliceStable(substitutions, func(i int, j int) bool {
					return substitutions[i]["file"] < substitutions[j]["file"]
				})
				Expect(substitutions).To(Equal(
					[]map[string]string{
						{
							"file":    "path/to/file1.js",
							"project": "project1",
							"grep":    `name of describe test '"'"'one'"'"' \+ 1`,
						},
					},
				))
			})
		})

		Describe("the new template format", func() {
			It("targets the annotated serial suite location", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test {{ tests }} --project '{{ project }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				fixture, err := os.Open("../../test/fixtures/playwright_serial.json")
				Expect(err).ToNot(HaveOccurred())
				testResults, err := parsing.JavaScriptPlaywrightParser{}.Parse(fixture)
				Expect(err).ToNot(HaveOccurred())

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					*testResults,
					func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(substitutions).To(Equal([]map[string]string{
					{
						"project": "chromium",
						"tests":   "/project/tests/example.spec.ts:10",
					},
				}))
			})

			It("targets the whole serial suite file when its line is zero", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test {{ tests }} --project '{{ project }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				testResults := v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     "serial failure",
							Location: &v1.Location{File: "tests/example.spec.ts", Line: new(42)},
							Attempt: v1.TestAttempt{
								Meta: map[string]any{
									"annotations": []parsing.JavaScriptPlaywrightAnnotation{
										{
											Type: "rwx:serial",
											Location: &parsing.JavaScriptPlaywrightLocation{
												File: "tests/example.spec.ts",
												Line: 0,
											},
										},
									},
									"project": "chromium",
								},
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					testResults,
					func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(substitutions).To(Equal([]map[string]string{
					{
						"project": "chromium",
						"tests":   "tests/example.spec.ts",
					},
				}))
			})

			It("keeps mixed ordinary and deduplicated serial targets", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test {{ tests }} --project '{{ project }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				failedTest := func(
					name string,
					line int,
					annotations []parsing.JavaScriptPlaywrightAnnotation,
				) v1.Test {
					return v1.Test{
						Name:     name,
						Location: &v1.Location{File: "tests/example.spec.ts", Line: &line},
						Attempt: v1.TestAttempt{
							Meta: map[string]any{
								"annotations": annotations,
								"project":     "chromium",
							},
							Status: v1.NewFailedTestStatus(nil, nil, nil),
						},
					}
				}
				serialAnnotation := func(line int) parsing.JavaScriptPlaywrightAnnotation {
					return parsing.JavaScriptPlaywrightAnnotation{
						Type: "rwx:serial",
						Location: &parsing.JavaScriptPlaywrightLocation{
							File: "tests/example.spec.ts",
							Line: line,
						},
					}
				}

				testResults := v1.TestResults{
					Tests: []v1.Test{
						failedTest("first serial failure", 41, []parsing.JavaScriptPlaywrightAnnotation{
							serialAnnotation(10),
						}),
						failedTest("second serial failure", 42, []parsing.JavaScriptPlaywrightAnnotation{
							serialAnnotation(10),
						}),
						failedTest("ordinary failure", 50, []parsing.JavaScriptPlaywrightAnnotation{
							{
								Type:     "issue",
								Location: &parsing.JavaScriptPlaywrightLocation{File: "tests/example.spec.ts", Line: 5},
							},
						}),
						failedTest("invalid serial annotation", 55, []parsing.JavaScriptPlaywrightAnnotation{
							{Type: "rwx:serial"},
						}),
						failedTest("whole file serial failure", 60, []parsing.JavaScriptPlaywrightAnnotation{
							serialAnnotation(0),
						}),
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					testResults,
					func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(substitutions).To(Equal([]map[string]string{
					{
						"project": "chromium",
						"tests": "tests/example.spec.ts:10 tests/example.spec.ts:50 " +
							"tests/example.spec.ts:55 tests/example.spec.ts",
					},
				}))
			})

			It("returns tests grouped by file and project", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test {{ tests }} --project '{{ project }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				file1 := "path/to/file1.js"
				file2 := "path/to/file with '.js"

				project1 := "project1"
				project2 := "project with ' 2"

				name1 := "name of describe test 'one' + 1"
				name2 := "name of describe test 2"

				lineOne := 1
				lineTen := 10

				testResults := v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     name1,
							Location: &v1.Location{File: file1, Line: &lineOne},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1, Line: &lineTen},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewCanceledTestStatus(),
							},
						},
						{
							Name:     name1,
							Location: &v1.Location{File: file2, Line: &lineTen},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewTimedOutTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewPendedTestStatus(nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewSuccessfulTestStatus(),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewSkippedTestStatus(nil),
							},
						},
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					testResults,
					func(t v1.Test) bool { return t.Attempt.Status.ImpliesFailure() },
				)
				Expect(err).NotTo(HaveOccurred())
				sort.SliceStable(substitutions, func(i int, j int) bool {
					return substitutions[i]["tests"] < substitutions[j]["tests"]
				})
				Expect(substitutions).To(Equal(
					[]map[string]string{
						{
							"project": "project with '\"'\"' 2",
							"tests":   "path/to/file with '\"'\"'.js:10",
						},
						{
							"project": "project1",
							"tests":   "path/to/file1.js:1 path/to/file1.js:10",
						},
					},
				))
			})

			It("filters the tests with the provided function", func() {
				compiledTemplate, compileErr := templating.CompileTemplate(
					"npx playwright test {{ tests }} --project '{{ project }}'",
				)
				Expect(compileErr).NotTo(HaveOccurred())

				file1 := "path/to/file1.js"
				file2 := "path/to/file with '.js"

				project1 := "project1"
				project2 := "project with ' 2"

				name1 := "name of describe test 'one' + 1"
				name2 := "name of describe test 2"

				lineOne := 1
				lineTen := 10

				testResults := v1.TestResults{
					Tests: []v1.Test{
						{
							Name:     name1,
							Location: &v1.Location{File: file1, Line: &lineOne},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewFailedTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1, Line: &lineTen},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewCanceledTestStatus(),
							},
						},
						{
							Name:     name1,
							Location: &v1.Location{File: file2, Line: &lineTen},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewTimedOutTestStatus(nil, nil, nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewPendedTestStatus(nil),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file1},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project2},
								Status: v1.NewSuccessfulTestStatus(),
							},
						},
						{
							Name:     name2,
							Location: &v1.Location{File: file2},
							Attempt: v1.TestAttempt{
								Meta:   map[string]any{"project": project1},
								Status: v1.NewSkippedTestStatus(nil),
							},
						},
					},
				}

				substitution := targetedretries.JavaScriptPlaywrightSubstitution{}
				substitutions, err := substitution.SubstitutionsFor(
					compiledTemplate,
					testResults,
					func(test v1.Test) bool { return test.Attempt.Status.Kind == v1.TestStatusFailed },
				)
				Expect(err).NotTo(HaveOccurred())
				sort.SliceStable(substitutions, func(i int, j int) bool {
					return substitutions[i]["tests"] < substitutions[j]["tests"]
				})
				Expect(substitutions).To(Equal(
					[]map[string]string{
						{
							"project": "project1",
							"tests":   "path/to/file1.js:1",
						},
					},
				))
			})
		})
	})
})
