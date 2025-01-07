package testresultsschema_test

import (
	"encoding/json"

	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestResults", func() {
	Describe("Marshal/UnmarshalJSON", func() {
		It("round trips", func() {
			testResults := v1.TestResults{
				Framework: v1.Framework{
					Language: "Ruby",
					Kind:     "RSpec",
				},
				Summary: v1.Summary{
					Status:      v1.SummaryStatusSuccessful,
					Tests:       1,
					OtherErrors: 2,
					Retries:     3,
					Canceled:    4,
					Failed:      5,
					Pended:      6,
					Quarantined: 7,
					Skipped:     8,
					Successful:  9,
					TimedOut:    10,
					Todo:        11,
				},
				Tests: []v1.Test{
					{
						Name: "name of the test",
						Attempt: v1.TestAttempt{
							Duration: nil,
							Status:   v1.TestStatus{Kind: "successful"},
						},
					},
				},
			}

			data, err := json.Marshal(testResults)

			Expect(err).To(BeNil())

			var parsedTestResults v1.TestResults
			err = json.Unmarshal([]byte(data), &parsedTestResults)

			Expect(err).To(BeNil())
			Expect(parsedTestResults).To(Equal(testResults))
		})
	})

	Describe("MarshalJSON", func() {
		It("includes the schema in the marshalled JSON", func() {
			json, err := json.Marshal(v1.TestResults{Summary: v1.Summary{Status: v1.SummaryStatusSuccessful}})

			Expect(err).To(BeNil())
			Expect(string(json)).To(
				ContainSubstring(
					`"$schema":"https://raw.githubusercontent.com/rwx-research/test-results-schema/main/v1.json"`,
				),
			)
		})

		It("marshals to a valid schema", func() {
			json, err := json.Marshal(
				v1.TestResults{
					Framework: v1.Framework{
						Language: "Ruby",
						Kind:     "RSpec",
					},
					Summary: v1.Summary{
						Status:      v1.SummaryStatusSuccessful,
						Tests:       1,
						OtherErrors: 2,
						Retries:     3,
						Canceled:    4,
						Failed:      5,
						Pended:      6,
						Quarantined: 7,
						Skipped:     8,
						Successful:  9,
						TimedOut:    10,
						Todo:        11,
					},
					Tests: []v1.Test{
						{
							Name: "name of the test",
							Attempt: v1.TestAttempt{
								Duration: nil,
								Status:   v1.TestStatus{Kind: "successful"},
							},
						},
					},
				},
			)

			Expect(err).To(BeNil())
			Expect(string(json)).To(ContainSubstring(`"framework":{"language":"Ruby","kind":"RSpec"}`))
			Expect(string(json)).To(ContainSubstring(`"summary":{"status":{"kind":"successful"}`))
			Expect(string(json)).To(
				ContainSubstring(`"tests":[{"name":"name of the test","attempt":{"durationInNanoseconds":null`),
			)
		})
	})
})
