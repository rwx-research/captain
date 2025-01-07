package parsing_test

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/bradleyjkemp/cupaloy"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JavaScriptKarmaParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/karma.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.JavaScriptKarmaParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on JSON that doesn't look like Karma", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptKarmaParser{}.Parse(strings.NewReader(`{
				"result": {},
				"summary": {}
			}
			`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The file does not look like a Karma file"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptKarmaParser{}.Parse(strings.NewReader(`{
				"browsers": {},
				"summary": {}
			}
			`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The file does not look like a Karma file"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptKarmaParser{}.Parse(strings.NewReader(`{
				"browsers": {},
				"result": {}
			}
			`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The file does not look like a Karma file"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.JavaScriptKarmaParser{}.Parse(strings.NewReader(`{
				"browsers": {
				},
				"result": {
					"87843490": [
						{
							"fullName": "Some test context is a contextual passing test",
							"description": "is a contextual passing test",
							"id": "spec3",
							"log": [],
							"skipped": false,
							"disabled": false,
							"pending": false,
							"success": true,
							"suite": ["Some test", "context"],
							"time": 1,
							"executedExpectationsCount": 1,
							"passedExpectations": [
								{
									"matcherName": "toBe",
									"message": "Passed.",
									"stack": "",
									"passed": true
								}
							],
							"properties": null
						}
					]
				},
				"summary": {
					"success": 2,
					"failed": 1,
					"skipped": 1,
					"error": true,
					"disconnected": false,
					"exitCode": 1
				}
			}
			`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The file does not look like a Karma file"))
			Expect(testResults).To(BeNil())
		})

		It("puts a generic other error in place when there are no errors but the suite failed", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.JavaScriptKarmaParser{}.Parse(strings.NewReader(`{
				"browsers": {
					"87843490": {
						"id": "87843490",
						"fullName": "Mozilla/5.0",
						"name": "Chrome 109.0.0.0 (Mac OS 10.15.7)",
						"state": "DISCONNECTED",
						"lastResult": {
							"startTime": 1677015045628,
							"total": 4,
							"success": 2,
							"failed": 1,
							"skipped": 1,
							"totalTime": 6,
							"netTime": 2,
							"error": true,
							"disconnected": false
						},
						"disconnectsCount": 0,
						"noActivityTimeout": 30000,
						"disconnectDelay": 2000
					}
				},
				"result": {
					"87843490": [
						{
							"fullName": "Some test context is a contextual passing test",
							"description": "is a contextual passing test",
							"id": "spec3",
							"log": [],
							"skipped": false,
							"disabled": false,
							"pending": false,
							"success": true,
							"suite": ["Some test", "context"],
							"time": 1,
							"executedExpectationsCount": 1,
							"passedExpectations": [
								{
									"matcherName": "toBe",
									"message": "Passed.",
									"stack": "",
									"passed": true
								}
							],
							"properties": null
						}
					]
				},
				"summary": {
					"success": 1,
					"failed": 0,
					"skipped": 0,
					"error": true,
					"disconnected": false,
					"exitCode": 1
				}
			}
			`))
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})
	})
})
