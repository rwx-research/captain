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

var _ = Describe("RubyRSpecParser", func() {
	Describe("Parse", func() {
		It("parses the sample file", func() {
			fixture, err := os.Open("../../test/fixtures/rspec.json")
			Expect(err).ToNot(HaveOccurred())

			testResults, err := parsing.RubyRSpecParser{}.Parse(fixture)
			Expect(err).ToNot(HaveOccurred())

			rwxJSON, err := json.MarshalIndent(testResults, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			cupaloy.SnapshotT(GinkgoT(), rwxJSON)
		})

		It("errors on malformed JSON", func() {
			testResults, err := parsing.RubyRSpecParser{}.Parse(strings.NewReader(`{"examples":[`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to parse test results as JSON"))
			Expect(testResults).To(BeNil())
		})

		It("errors JSON that doesn't look like RSpec", func() {
			var testResults *v1.TestResults
			var err error

			testResults, err = parsing.RubyRSpecParser{}.Parse(strings.NewReader(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No examples were found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.RubyRSpecParser{}.Parse(strings.NewReader(`{"examples": []}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No summary was found in the JSON"))
			Expect(testResults).To(BeNil())

			testResults, err = parsing.RubyRSpecParser{}.Parse(
				strings.NewReader(`{"examples": [{}], "summary": {}}`),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The examples in the JSON do not appear to match RSpec JSON"))
			Expect(testResults).To(BeNil())
		})

		It("parses examples that failed", func() {
			id := "./spec/some/id_spec.rb:[1:2:3]"
			rspecResults := parsing.RubyRSpecTestResults{
				Examples: []parsing.RubyRSpecExample{
					{
						ID:              &id,
						Description:     "the test description",
						FullDescription: "Some::Class the test description",
						Status:          "failed",
						FilePath:        "./spec/some/other/file/path_spec.rb",
						LineNumber:      12,
						RunTime:         0.00005,
						PendingMessage:  nil,
						Exception: &parsing.RubyRSpecException{
							Class:     "ExceptionClass",
							Message:   "ExceptionMessage",
							Backtrace: []string{"1", "2", "3"},
						},
					},
				},
				Summary: &parsing.RubyRSpecSummary{},
			}
			data, err := json.Marshal(rspecResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.RubyRSpecParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(50000)
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					ID:       &id,
					Name:     "Some::Class the test description",
					Lineage:  []string{"Some::Class", "the test description"},
					Location: &v1.Location{File: "./spec/some/id_spec.rb"},
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta:     map[string]any{"filePath": "./spec/some/other/file/path_spec.rb", "lineNumber": 12},
						Status: v1.NewFailedTestStatus(
							&rspecResults.Examples[0].Exception.Message,
							&rspecResults.Examples[0].Exception.Class,
							[]string{"1", "2", "3"},
						),
					},
				},
			))
		})

		It("parses successful examples", func() {
			rspecResults := parsing.RubyRSpecTestResults{
				Examples: []parsing.RubyRSpecExample{
					{
						Description:     "the test description",
						FullDescription: "Some::Class the test description",
						Status:          "passed",
						FilePath:        "./spec/some/other/file/path_spec.rb",
						LineNumber:      12,
						RunTime:         60.0,
						PendingMessage:  nil,
					},
				},
				Summary: &parsing.RubyRSpecSummary{},
			}
			data, err := json.Marshal(rspecResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.RubyRSpecParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(60000000000)
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					Name:     "Some::Class the test description",
					Lineage:  []string{"Some::Class", "the test description"},
					Location: &v1.Location{File: "./spec/some/other/file/path_spec.rb"},
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta:     map[string]any{"filePath": "./spec/some/other/file/path_spec.rb", "lineNumber": 12},
						Status:   v1.NewSuccessfulTestStatus(),
					},
				},
			))
		})

		It("parses pending examples", func() {
			id := "./spec/some/id_spec.rb:[1:2:3]"
			pendingMessage := "pending message"
			rspecResults := parsing.RubyRSpecTestResults{
				Examples: []parsing.RubyRSpecExample{
					{
						ID:              &id,
						Description:     "the test description",
						FullDescription: "Some::Class the test description",
						Status:          "pending",
						FilePath:        "./spec/some/other/file/path_spec.rb",
						LineNumber:      12,
						RunTime:         60.0,
						PendingMessage:  &pendingMessage,
					},
				},
				Summary: &parsing.RubyRSpecSummary{},
			}
			data, err := json.Marshal(rspecResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.RubyRSpecParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).NotTo(HaveOccurred())
			Expect(testResults).NotTo(BeNil())

			duration := time.Duration(60000000000)
			Expect(testResults.Tests[0]).To(Equal(
				v1.Test{
					ID:       &id,
					Name:     "Some::Class the test description",
					Lineage:  []string{"Some::Class", "the test description"},
					Location: &v1.Location{File: "./spec/some/id_spec.rb"},
					Attempt: v1.TestAttempt{
						Duration: &duration,
						Meta:     map[string]any{"filePath": "./spec/some/other/file/path_spec.rb", "lineNumber": 12},
						Status:   v1.NewPendedTestStatus(&pendingMessage),
					},
				},
			))
		})

		It("fails to parse other statuses", func() {
			id := "./spec/some/id_spec.rb:[1:2:3]"
			rspecResults := parsing.RubyRSpecTestResults{
				Examples: []parsing.RubyRSpecExample{
					{
						ID:              &id,
						Description:     "the test description",
						FullDescription: "Some::Class the test description",
						Status:          "wat",
						FilePath:        "./spec/some/other/file/path_spec.rb",
						LineNumber:      12,
						RunTime:         60.0,
						PendingMessage:  nil,
					},
				},
				Summary: &parsing.RubyRSpecSummary{},
			}
			data, err := json.Marshal(rspecResults)
			Expect(err).NotTo(HaveOccurred())

			testResults, err := parsing.RubyRSpecParser{}.Parse(strings.NewReader(string(data)))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unexpected status \"wat\""))
			Expect(testResults).To(BeNil())
		})
	})
})
