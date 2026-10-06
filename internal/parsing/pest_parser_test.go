package parsing_test

import (
	"strings"
	"time"

	"github.com/rwx-research/captain-cli/internal/parsing"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PestParser", func() {
	It("normalizes logical locations without changing test identity descriptions", func() {
		results, err := (parsing.PestParser{}).Parse(strings.NewReader(`<testsuites>
  <testsuite name="Tests\ArchTest" file="tests/ArchTest.php">
    <testcase name="controllers :: use interfaces" class="Tests\ArchTest"
      file="tests/ArchTest.php::controllers :: use interfaces" time="1.25"/>
    <testsuite name="uses contracts" file="tests/ArchTest.php::uses contracts">
      <testcase name="uses contracts with data set &quot;(true)&quot;" class="Tests\ArchTest"
        file="tests/ArchTest.php::uses contracts with data set &quot;(true)&quot;" time="2.5">
        <failure type="ExpectationFailed">failed</failure>
      </testcase>
    </testsuite>
  </testsuite>
</testsuites>`))
		Expect(err).NotTo(HaveOccurred())
		Expect(results.Framework).To(Equal(v1.PHPPestFramework))
		Expect(results.Tests).To(HaveLen(2))
		Expect(results.Tests[0].Location.File).To(Equal("tests/ArchTest.php"))
		Expect(results.Tests[1].Location.File).To(Equal("tests/ArchTest.php"))
		Expect(results.Tests[0].Name).To(Equal(`Tests\ArchTest::controllers :: use interfaces`))
		Expect(results.Tests[0].Lineage).To(Equal([]string{`Tests\ArchTest`, "controllers :: use interfaces"}))
		Expect(results.Tests[0].Attempt.Meta).NotTo(HaveKey("pestDatasetName"))
		Expect(results.Tests[1].Attempt.Meta).To(HaveKeyWithValue("pestDatasetName", "uses contracts"))
		Expect(*results.Tests[0].Attempt.Duration + *results.Tests[1].Attempt.Duration).To(Equal(3750 * time.Millisecond))
		Expect(results.Tests[1].Attempt.Status.Kind).To(Equal(v1.TestStatusFailed))
	})

	It("recovers early Pest 3 dataset descriptions without truncating dataset-like text", func() {
		results, err := (parsing.PestParser{}).Parse(strings.NewReader(`<testsuites>
  <testsuite name="Tests\ExampleTest::__pest_evaluable_uses__data_with_data_set_literals">
    <testcase name="uses_data with data set literals with data set &quot;(true)&quot;" class="Tests\ExampleTest"
      file="tests/ExampleTest.php::uses_data with data set literals with data set &quot;(true)&quot;" time="0.1"/>
  </testsuite>
</testsuites>`))
		Expect(err).NotTo(HaveOccurred())
		Expect(results.Tests[0].Attempt.Meta).To(HaveKeyWithValue("pestDatasetName", "uses_data with data set literals"))
	})

	It("preserves physical paths and descriptions containing dataset-like text", func() {
		results, err := (parsing.PestParser{}).Parse(strings.NewReader(`<testsuites>
  <testsuite name="Tests\ExampleTest">
    <testcase name="works with data set examples" class="Tests\ExampleTest"
      file="C:\project\tests\ExampleTest.php" line="42" time="0.1"/>
  </testsuite>
</testsuites>`))
		Expect(err).NotTo(HaveOccurred())
		Expect(results.Tests[0].Location.File).To(Equal(`C:\project\tests\ExampleTest.php`))
		Expect(*results.Tests[0].Location.Line).To(Equal(42))
		Expect(results.Tests[0].Attempt.Meta).NotTo(HaveKey("pestDatasetName"))
	})

	It("propagates invalid XML errors", func() {
		results, err := (parsing.PestParser{}).Parse(strings.NewReader(`<invalid`))
		Expect(err).To(HaveOccurred())
		Expect(results).To(BeNil())
	})
})
