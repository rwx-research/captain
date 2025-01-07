package parsing_test

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptCypressParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed XML", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<abc`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())
		})

		It("errors on XML that doesn't look like Cypress", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<foo></foo>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(`<testsuites></testsuites>`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No tests count was found in the XML"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptCypressParser{}.Parse(
				strings.NewReader(`<testsuites tests="1"></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match Cypress XML"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptCypressParser{}.Parse(
				strings.NewReader(`<testsuites tests="1"><testsuite></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The test suites in the XML do not appear to match Cypress XML"),
			)
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptCypressParser{}.Parse(
				strings.NewReader(`<testsuites tests="1"><testsuite file="not/right.js"></testsuite></testsuites>`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring("The file does not look like a Cypress file: \"not/right.js\""),
			)
			Expect(testResults).To(BeNil())
		})

		It("parses files properly", func() {
			fixture, err := os.Open("../../test/fixtures/cypress.xml")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			nameOne := "example to-do app one displays two todo items by default"
			nameTwo := "example to-do app two can add new todo items"
			for _, test := range testResults.Tests {
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

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			name := "displays two todo items by default"
			for _, test := range testResults.Tests {
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

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			name := "example to-do app one displays two todo items by default"
			for _, test := range testResults.Tests {
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

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			name := "example to-do app one can add new todo items"
			for _, test := range testResults.Tests {
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

			testResults, err := parsing.JavaScriptCypressParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			name := "example to-do app one with a checked task can filter for uncompleted tasks"
			for _, test := range testResults.Tests {
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
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).To(BeNil())
		})

		It("errors when the file name doesn't look like Cypress", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).To(BeNil())
		})

		It("calculates the correct name when the classname contains the name", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name contains the classname", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is the same as classname", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})

		It("calculates the correct name when the name is entirely different from the classname", func() {
			testResults, err := parsing.JavaScriptCypressParser{}.Parse(strings.NewReader(
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
			Expect(testResults).NotTo(BeNil())
			Expect(testResults.Tests[0].Name).To(Equal("prefix some test name"))
		})
	})
})
