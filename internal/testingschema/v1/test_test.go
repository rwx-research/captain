package v1_test

import (
	"encoding/json"
	"time"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestAttempt", func() {
	It("serializes the duration to nanoseconds", func() {
		duration := time.Duration(1500000000)
		json, err := json.Marshal(v1.TestAttempt{Duration: &duration})

		Expect(err).To(BeNil())
		Expect(string(json)).To(ContainSubstring(`"durationInNanoseconds":1500000000`))
	})

	It("serializes no duration to null", func() {
		json, err := json.Marshal(v1.TestAttempt{Duration: nil})

		Expect(err).To(BeNil())
		Expect(string(json)).To(ContainSubstring(`"durationInNanoseconds":null`))
	})
})
