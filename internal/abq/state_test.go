package abq_test

import (
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/abq"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("State", func() {
	It("parses from json", func() {
		contents := `{
      "abq_version": "1.1.2",
      "abq_executable": "/path/to/abq",
      "run_id": "not-a-real-run-id",
      "supervisor": true
    }`

		var state abq.State
		err := json.Unmarshal([]byte(contents), &state)
		Expect(err).ToNot(HaveOccurred())

		Expect(state.AbqVersion).To(Equal("1.1.2"))
		Expect(state.AbqExecutable).To(Equal("/path/to/abq"))
		Expect(state.RunID).To(Equal("not-a-real-run-id"))
		Expect(state.Supervisor).To(BeTrue())
	})
})
