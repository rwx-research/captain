package parsing_test

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DotNetxUnitParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/xunit_dot_net.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.DotNetxUnitParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like xUnit.NET", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.DotNetxUnitParser{}.Parse(strings.NewReader(`<assemblies></assemblies>`))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			testResults, err = parsing.DotNetxUnitParser{}.Parse(
				strings.NewReader(`<assemblies><assembly></assembly></assemblies>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match xUnit.NET XML"),
			)
			Expect(testResults).To(BeNil())
		})

		It("can extract a detailed successful test", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									id="some-id"
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="Pass"
								>
									<source-file>some/path/to/source.cs</source-file>
									<source-line>12</source-line>
									<traits>
										<trait name="some-trait" value="some-value" />
										<trait name="other-trait" value="other-value" />
									</traits>
									<output><![CDATA[line 1
line 2
line 3]]></output>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			line := 12
			id := "some-id"
			duration := time.Duration(6370900)
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			stdout := "line 1\nline 2\nline 3"
			assembly := "AssemblyName.dll"
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Scope:    &assembly,
					ID:       &id,
					Name:     "NullAssertsTests+Null.Success",
					Location: &v1.Location{File: "some/path/to/source.cs", Line: &line},
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly":          assembly,
							"type":              testType,
							"method":            testMethod,
							"trait-some-trait":  "some-value",
							"trait-other-trait": "other-value",
						},
						Status: v1.NewSuccessfulTestStatus(),
						Stdout: &stdout,
					},
				},
			))
		})

		It("can extract a failed test", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="Fail"
								>
									<failure exception-type="AssertionException">
										<message><![CDATA[Some message here]]></message>
										<stack-trace><![CDATA[Some trace
											other line]]></stack-trace>
									</failure>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(6370900)
			message := "Some message here"
			exception := "AssertionException"
			assembly := "AssemblyName.dll"
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Scope: &assembly,
					Name:  "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": assembly,
							"type":     testType,
							"method":   testMethod,
						},
						Status: v1.NewFailedTestStatus(&message, &exception, []string{"Some trace", "other line"}),
					},
				},
			))
		})

		It("can extract a failed test without details", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="Fail"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(6370900)
			assembly := "AssemblyName.dll"
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Scope: &assembly,
					Name:  "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": assembly,
							"type":     testType,
							"method":   testMethod,
						},
						Status: v1.NewFailedTestStatus(nil, nil, nil),
					},
				},
			))
		})

		It("can extract a skipped test", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="Skip"
								>
									<reason><![CDATA[Some reason here]]></reason>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(6370900)
			message := "Some reason here"
			assembly := "AssemblyName.dll"
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Scope: &assembly,
					Name:  "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": assembly,
							"type":     testType,
							"method":   testMethod,
						},
						Status: v1.NewSkippedTestStatus(&message),
					},
				},
			))
		})

		It("can extract a not-run test", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="NotRun"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(6370900)
			assembly := "AssemblyName.dll"
			testType := "NullAssertsTests+Null"
			testMethod := "Success"
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Scope: &assembly,
					Name:  "NullAssertsTests+Null.Success",
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta: map[string]any{
							"assembly": assembly,
							"type":     testType,
							"method":   testMethod,
						},
						Status: v1.NewSkippedTestStatus(nil),
					},
				},
			))
		})

		It("errors on other results", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<collection>
								<test
									name="NullAssertsTests+Null.Success"
									type="NullAssertsTests+Null"
									method="Success"
									time="0.0063709"
									result="wat"
								>
								</test>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`Unexpected result "wat"`))
			Expect(testResults).To(BeNil())
		})

		It("can extract other errors", func() {
			testResults, err := parsing.DotNetxUnitParser{}.Parse(strings.NewReader(
				`
					<assemblies>
						<assembly name="some/path/to/AssemblyName.dll">
							<errors>
								<error name="ErrorName" type="error-type-one">
									<failure exception-type="SomeException">
										<message><![CDATA[Some message here]]></message>
										<stack-trace><![CDATA[Some trace
											other line]]></stack-trace>
									</failure>
								</error>
								<error type="error-type-two">
									<failure />
								</error>
							</errors>
							<collection>
							</collection>
						</assembly>
					</assemblies>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())
			exception := "SomeException"
			Expect(testResults.OtherErrors).To(Equal(
				[]v1.OtherError{
					{
						Backtrace: []string{"Some trace", "other line"},
						Exception: &exception,
						Message:   "Some message here",
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     "error-type-one",
						},
					},
					{
						Message: "An error occurred during error-type-two",
						Meta: map[string]any{
							"assembly": "AssemblyName.dll",
							"type":     "error-type-two",
						},
					},
				},
			))
		})
	})
})
