package v1_test

import (
	"encoding/json"

	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SummaryStatus", func() {
	It("marshals the summary statuses", func() {
		var data []byte
		var err error

		data, err = json.Marshal(v1.SummaryStatusSuccessful)

		Expect(err).To(BeNil())
		Expect(string(data)).To(Equal(`{"kind":"successful"}`))

		data, err = json.Marshal(v1.SummaryStatusFailed)

		Expect(err).To(BeNil())
		Expect(string(data)).To(Equal(`{"kind":"failed"}`))

		data, err = json.Marshal(v1.SummaryStatusCanceled)

		Expect(err).To(BeNil())
		Expect(string(data)).To(Equal(`{"kind":"canceled"}`))

		data, err = json.Marshal(v1.SummaryStatusTimedOut)

		Expect(err).To(BeNil())
		Expect(string(data)).To(Equal(`{"kind":"timedOut"}`))

		data, err = json.Marshal(v1.SummaryStatus("foo"))

		Expect(err).To(BeNil())
		Expect(string(data)).To(Equal(`{"kind":"foo"}`))
	})

	It("unmarshals the summary statuses", func() {
		var ss v1.SummaryStatus
		var err error
		var data string

		data = `{"kind":"successful"}`
		err = json.Unmarshal([]byte(data), &ss)

		Expect(err).To(BeNil())
		Expect(ss).To(Equal(v1.SummaryStatusSuccessful))

		data = `{"kind":"failed"}`
		err = json.Unmarshal([]byte(data), &ss)

		Expect(err).To(BeNil())
		Expect(ss).To(Equal(v1.SummaryStatusFailed))

		data = `{"kind":"canceled"}`
		err = json.Unmarshal([]byte(data), &ss)

		Expect(err).To(BeNil())
		Expect(ss).To(Equal(v1.SummaryStatusCanceled))

		data = `{"kind":"timedOut"}`
		err = json.Unmarshal([]byte(data), &ss)

		Expect(err).To(BeNil())
		Expect(ss).To(Equal(v1.SummaryStatusTimedOut))

		data = `{"kind":"foo"}`
		err = json.Unmarshal([]byte(data), &ss)

		Expect(err).To(BeNil())
		Expect(ss).To(Equal(v1.SummaryStatus("foo")))
	})
})
