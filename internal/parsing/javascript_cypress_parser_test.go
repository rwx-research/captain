package parsing_test

import (
	"os"
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptCypressParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			Expect(parseResult.Parser).To(Equal(parsing.JavaScriptCypressParser{}))
			Expect(parseResult.Sentiment).To(Equal(parsing.PositiveParseResultSentiment))
			Expect(parseResult.TestResults.Framework.Language).To(Equal(v1.FrameworkLanguageJavaScript))
			Expect(parseResult.TestResults.Framework.Kind).To(Equal(v1.FrameworkKindCypress))
			Expect(parseResult.TestResults.Summary.Tests).To(Equal(19))
			Expect(parseResult.TestResults.Summary.Successful).To(Equal(14))
			Expect(parseResult.TestResults.Summary.Failed).To(Equal(3))
			Expect(parseResult.TestResults.Summary.Skipped).To(Equal(2))
			Expect(parseResult.TestResults.Summary.OtherErrors).To(Equal(0))
		})

		It("errors on malformed XML", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())
		})

		It("errors on XML that doesn't look like Cypress", func() {
			var parseResult *parsing.ParseResult
			var err error

			parseResult, err = parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No tests count was found in the XML"))
			Expect(parseResult).To(BeNil())

			parseResult, err = parsing.JavaScriptCypressParser{}.Parse(
				strings.NewReader(`<testsuites tests="1"><testsuite></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match Cypress XML"),
			)
			Expect(parseResult).To(BeNil())
		})

		It("parses files properly", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			nameOne := "example to-do app one displays two todo items by default"
			nameTwo := "example to-do app two can add new todo items"
			for _, test := range parseResult.TestResults.Tests {
				switch test.Name {
				case nameOne:
					Expect(test.Location.File).To(Equal("cypress/e2e/todo-one.cy.js"))
				case nameTwo:
					Expect(test.Location.File).To(Equal("cypress/e2e/todo-two.cy.js"))
				default:
					continue
				}
				return
			}

			Fail("Unreachable")
		})

		It("parses successful tests", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			name := "displays two todo items by default"
			for _, test := range parseResult.TestResults.Tests {
				if test.Name != name {
					continue
				}

				Expect(test.Attempt.Status).To(Equal(v1.NewSuccessfulTestStatus()))
				Expect(*test.Attempt.Duration).To(Equal(time.Duration(490000000)))

				return
			}

			Fail("Unreachable")
		})

		It("parses tests which failed via an assertion", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			name := "example to-do app one displays two todo items by default"
			for _, test := range parseResult.TestResults.Tests {
				if test.Name != name {
					continue
				}

				message := "Timed out retrying after 4000ms: expected '<li>' to have text 'Walk the cat', but" +
					" the text was 'Walk the dog'"
				class := "AssertionError"
				backtrace := []string{
					"at Context.eval (webpack:///./cypress/e2e/todo-one.cy.js:59:35)",
					"",
					"+ expected - actual",
					"",
					"-'Walk the dog'",
					"+'Walk the cat'",
					"",
				}
				Expect(test.Attempt.Status).To(Equal(v1.NewFailedTestStatus(&message, &class, backtrace)))

				return
			}

			Fail("Unreachable")
		})

		It("parses tests which failed via a custom error", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			name := "example to-do app one can add new todo items"
			for _, test := range parseResult.TestResults.Tests {
				if test.Name != name {
					continue
				}

				message := "uh oh something broke"
				class := "Error"
				backtrace := []string{"at Context.eval (webpack:///./cypress/e2e/todo-one.cy.js:73:10)"}
				Expect(test.Attempt.Status).To(Equal(v1.NewFailedTestStatus(&message, &class, backtrace)))

				return
			}

			Fail("Unreachable")
		})

		It("parses tests which failed via a custom error", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())

			name := "example to-do app one with a checked task can filter for uncompleted tasks"
			for _, test := range parseResult.TestResults.Tests {
				if test.Name != name {
					continue
				}

				message := "not a custom error"
				class := "Error"
				backtrace := []string{
					"at $Cy.fail (https://example.cypress.io/__cypress/runner/cypress_runner.js:157093:13)",
					"at runnable.fn (https://example.cypress.io/__cypress/runner/cypress_runner.js:157707:19)",
					"at callFn (https://example.cypress.io/__cypress/runner/cypress_runner.js:107910:21)",
					"at ../driver/node_modules/mocha/lib/runnable.js.Runnable.run" +
						" (https://example.cypress.io/__cypress/runner/cypress_runner.js:107897:7)",
					"at <unknown> (https://example.cypress.io/__cypress/runner/cypress_runner.js:164958:30)",
					"at PassThroughHandlerContext.finallyHandler" +
						" (https://example.cypress.io/__cypress/runner/cypress_runner.js:7872:23)",
					"at PassThroughHandlerContext.tryCatcher" +
						" (https://example.cypress.io/__cypress/runner/cypress_runner.js:11318:23)",
					"at Promise._settlePromiseFromHandler (https://example.cypress.io/__cypress/runner/cypress_runner.js:9253:31)",
					"at Promise._settlePromise (https://example.cypress.io/__cypress/runner/cypress_runner.js:9310:18)",
					"at Promise._settlePromise0 (https://example.cypress.io/__cypress/runner/cypress_runner.js:9355:10)",
					"at Promise._settlePromises (https://example.cypress.io/__cypress/runner/cypress_runner.js:9435:18)",
					"at _drainQueueStep (https://example.cypress.io/__cypress/runner/cypress_runner.js:6025:12)",
					"at _drainQueue (https://example.cypress.io/__cypress/runner/cypress_runner.js:6018:9)",
					"at ../../node_modules/bluebird/js/release/async.js.Async._drainQueues" +
						" (https://example.cypress.io/__cypress/runner/cypress_runner.js:6034:5)",
					"at Async.drainQueues (https://example.cypress.io/__cypress/runner/cypress_runner.js:5904:14)",
				}
				Expect(test.Attempt.Status).To(Equal(v1.NewFailedTestStatus(&message, &class, backtrace)))

				return
			}

			Fail("Unreachable")
		})

		It("errors when presented with an <error> element", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="cypress/e2e/some.test.cy.js">
							<testcase name="some test name">
								<error />
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`Unexpected <error> element in "some test name", mocha-junit-reporter does not emit these.`,
			))
			Expect(parseResult).To(BeNil())
		})

		It("errors when the file name doesn't look like Cypress", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="e2e/some.test.js">
							<testcase name="some test name">
								<error />
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`The file does not look like a Cypress file: "e2e/some.test.js"`,
			))
			Expect(parseResult).To(BeNil())
		})

		It("calculates the correct name when the classname contains the name", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="cypress/e2e/some.test.cy.js">
							<testcase name="some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name contains the classname", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="cypress/e2e/some.test.cy.js">
							<testcase name="prefix some test name" classname="some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is the same as classname", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="cypress/e2e/some.test.cy.js">
							<testcase name="prefix some test name" classname="prefix some test name">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is entirely different from the classname", func() {
			parseResult, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
				`
					<testsuites tests="1">
						<testsuite file="cypress/e2e/some.test.cy.js">
							<testcase name="some test name" classname="prefix">
							</testcase>
						</testsuite>
					</testsuites>
				`,
			))
			Expect(err).NotTo(HaveOccurred())
			Expect(parseResult).NotTo(BeNil())
			Expect(parseResult.TestResults.Tests[0].Name).To(Equal("prefix some test name"))
		})
	})
})
