package reporting

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/acarl005/stripansi"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/targetedretries"
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/pkg/testresultsschema/v1"
)

type markdownTestSection string

var (
	flakySection       markdownTestSection = "🔁 Flaky"
	failedSection      markdownTestSection = "❌ Failed"
	timedOutSection    markdownTestSection = "⏳ Timed Out"
	quarantinedSection markdownTestSection = "🏥 Quarantined"
	canceledSection    markdownTestSection = "🚫 Canceled"
)

type markdownTest struct {
	Name      string
	Location  string
	Command   string
	Message   *string
	Backtrace string
	Retries   int
}

const (
	oneMB                    = 1000000
	markdownResultsTruncated = "\n\nYour results have been truncated; markdown summarization has a 1MB limit."
	markdownTestTemplate     = `<details>
<summary><strong>{{ .Name }}</strong></summary>

<dl>
{{ if .Retries }}<dd>Retried {{ .Retries}} time{{ if ne .Retries 1 }}s{{end}}</dd>{{ end }}
{{ if .Location }}<dd>Defined at <code>{{ .Location }}</code></dd>{{ end }}
{{ if .Command }}<dd>Retry with <code>{{ .Command }}</code></dd>{{ end }}
{{ if or .Message .Backtrace }}
<dd>
<details>
<summary>Failure Details</summary><br />
{{ if and .Message .Backtrace }}
<pre>{{ .Message}}

{{ .Backtrace }}</pre>
{{ else }}
<pre>{{ or .Message .Backtrace }}</pre>
{{ end }}
</details>
</dd>
{{ end }}
</dl>
</details>
`
)

func WriteMarkdownSummary(file fs.File, testResults v1.TestResults, cfg Configuration) error {
	markdown := new(strings.Builder)
	if _, err := markdown.WriteString(fmt.Sprintf("# `%v` Summary\n\n", cfg.SuiteID)); err != nil {
		return errors.WithStack(err)
	}

	if cfg.CloudEnabled {
		if _, err := markdown.WriteString(
			fmt.Sprintf(
				"[🔗 View in Captain Cloud](https://%v/captain/%v/test_suite_summaries/%v/%v/%v)\n\n",
				cfg.CloudHost,
				cfg.CloudOrganizationSlug,
				cfg.SuiteID,
				cfg.Provider.BranchName,
				cfg.Provider.CommitSha,
			),
		); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := writeMarkdownSummaryLine(markdown, testResults); err != nil {
		return errors.WithStack(err)
	}

	testsBySection := testsByMarkdownSection(testResults)
	writersBySection := map[markdownTestSection]func(
		*strings.Builder,
		v1.Framework,
		[]v1.Test,
		Configuration,
	) (bool, error){
		flakySection:       writeMarkdownFlakySection,
		failedSection:      writeMarkdownFailedSection,
		timedOutSection:    writeMarkdownTimedOutSection,
		quarantinedSection: writeMarkdownQuarantinedSection,
		canceledSection:    writeMarkdownCanceledSection,
	}

	orderedSections := []markdownTestSection{
		flakySection,
		failedSection,
		timedOutSection,
		quarantinedSection,
		canceledSection,
	}

	for _, section := range orderedSections {
		shouldTruncate, err := writersBySection[section](markdown, testResults.Framework, testsBySection[section], cfg)
		if err != nil {
			return errors.WithStack(err)
		}
		if shouldTruncate {
			if _, err := markdown.WriteString(markdownResultsTruncated); err != nil {
				return errors.WithStack(err)
			}
			if _, err := file.Write([]byte(markdown.String())); err != nil {
				return errors.WithStack(err)
			}
			return nil
		}
	}

	if _, err := file.Write([]byte(markdown.String())); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func writeMarkdownSummaryStatus(markdown *strings.Builder, value int, singular string, plural string) error {
	if value <= 0 {
		return nil
	}

	if _, err := markdown.WriteString(fmt.Sprintf(", %v %v", value, pluralize(value, singular, plural))); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func writeMarkdownSummaryLine(markdown *strings.Builder, testResults v1.TestResults) error {
	summary := testResults.Summary

	if _, err := markdown.WriteString(
		fmt.Sprintf("%v %v", summary.Tests, pluralize(summary.Tests, "test", "tests")),
	); err != nil {
		return errors.WithStack(err)
	}

	flaky := 0
	for _, test := range testResults.Tests {
		if test.Flaky() {
			flaky++
		}
	}

	if err := writeMarkdownSummaryStatus(markdown, flaky, "flaky", "flaky"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Failed, "failed", "failed"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.TimedOut, "timed out", "timed out"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Quarantined, "quarantined", "quarantined"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Canceled, "canceled", "canceled"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Pended, "pended", "pended"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Skipped, "skipped", "skipped"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Todo, "todo", "todo"); err != nil {
		return errors.WithStack(err)
	}

	if err := writeMarkdownSummaryStatus(markdown, summary.Retries, "retry", "retries"); err != nil {
		return errors.WithStack(err)
	}

	if _, err := markdown.WriteString("\n"); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func testsByMarkdownSection(testResults v1.TestResults) map[markdownTestSection][]v1.Test {
	testsBySection := map[markdownTestSection][]v1.Test{
		flakySection:       make([]v1.Test, 0),
		failedSection:      make([]v1.Test, 0),
		timedOutSection:    make([]v1.Test, 0),
		quarantinedSection: make([]v1.Test, 0),
		canceledSection:    make([]v1.Test, 0),
	}

	for _, test := range testResults.Tests {
		// Flaky first so that anything that's flaky will end up only in that section.
		// The rest are mutually exclusive
		if test.Flaky() {
			testsBySection[flakySection] = append(testsBySection[flakySection], test)
			continue
		}

		if test.Attempt.Status.Kind == v1.TestStatusFailed {
			testsBySection[failedSection] = append(testsBySection[failedSection], test)
			continue
		}

		if test.Attempt.Status.Kind == v1.TestStatusTimedOut {
			testsBySection[timedOutSection] = append(testsBySection[timedOutSection], test)
			continue
		}

		if test.Attempt.Status.Kind == v1.TestStatusQuarantined {
			testsBySection[quarantinedSection] = append(testsBySection[quarantinedSection], test)
			continue
		}

		if test.Attempt.Status.Kind == v1.TestStatusCanceled {
			testsBySection[canceledSection] = append(testsBySection[canceledSection], test)
			continue
		}
	}

	return testsBySection
}

func writeMarkdownFlakySection(
	markdown *strings.Builder,
	framework v1.Framework,
	tests []v1.Test,
	cfg Configuration,
) (bool, error) {
	return writeMarkdownSection(
		markdown,
		flakySection,
		framework,
		tests,
		func(test v1.Test) *v1.TestStatus {
			if test.Attempt.Status.PotentiallyFlaky() {
				return &test.Attempt.Status
			}

			for _, attempt := range test.PastAttempts {
				if attempt.Status.PotentiallyFlaky() {
					return &attempt.Status
				}
			}

			return nil
		},
		cfg,
	)
}

func writeMarkdownFailedSection(
	markdown *strings.Builder,
	framework v1.Framework,
	tests []v1.Test,
	cfg Configuration,
) (bool, error) {
	return writeMarkdownSection(
		markdown,
		failedSection,
		framework,
		tests,
		func(test v1.Test) *v1.TestStatus {
			return &test.Attempt.Status
		},
		cfg,
	)
}

func writeMarkdownTimedOutSection(
	markdown *strings.Builder,
	framework v1.Framework,
	tests []v1.Test,
	cfg Configuration,
) (bool, error) {
	return writeMarkdownSection(
		markdown,
		timedOutSection,
		framework,
		tests,
		func(test v1.Test) *v1.TestStatus {
			return &test.Attempt.Status
		},
		cfg,
	)
}

func writeMarkdownQuarantinedSection(
	markdown *strings.Builder,
	framework v1.Framework,
	tests []v1.Test,
	cfg Configuration,
) (bool, error) {
	return writeMarkdownSection(
		markdown,
		quarantinedSection,
		framework,
		tests,
		func(test v1.Test) *v1.TestStatus {
			return test.Attempt.Status.OriginalStatus
		},
		cfg,
	)
}

func writeMarkdownCanceledSection(
	markdown *strings.Builder,
	framework v1.Framework,
	tests []v1.Test,
	cfg Configuration,
) (bool, error) {
	return writeMarkdownSection(
		markdown,
		canceledSection,
		framework,
		tests,
		func(test v1.Test) *v1.TestStatus {
			return &test.Attempt.Status
		},
		cfg,
	)
}

func writeMarkdownSection(
	markdown *strings.Builder,
	section markdownTestSection,
	framework v1.Framework,
	tests []v1.Test,
	findFailedStatus func(v1.Test) *v1.TestStatus,
	cfg Configuration,
) (bool, error) {
	if len(tests) == 0 {
		return false, nil
	}

	if _, err := markdown.WriteString(fmt.Sprintf("\n## %v\n\n", section)); err != nil {
		return false, errors.WithStack(err)
	}

	parsedTemplate, err := template.New("markdownTestTemplate").Parse(markdownTestTemplate)
	if err != nil {
		return false, errors.WithStack(err)
	}

	retryTemplate, substitution := retryTemplateAndSubstitutionFor(framework, cfg.RetryCommandTemplate)

	for _, test := range tests {
		location := ""
		if test.Location != nil {
			location = test.Location.String()
		}

		retryCommand := ""
		if retryTemplate != nil && substitution != nil {
			substitutions, _ := substitution.SubstitutionsFor(
				*retryTemplate,
				*v1.NewTestResults(framework, []v1.Test{test}, nil),
				func(_ v1.Test) bool { return true },
			)

			if len(substitutions) >= 1 {
				retryCommand = retryTemplate.Substitute(substitutions[0])
			}
		}
		failedStatus := findFailedStatus(test)
		markdownTest := markdownTest{
			Name:     test.Name,
			Location: location,
			Command:  retryCommand,
			Retries:  len(test.PastAttempts),
		}
		if failedStatus != nil {
			markdownTest.Backtrace = stripansi.Strip(strings.Join(failedStatus.Backtrace, "\n"))
			if failedStatus.Message != nil {
				strippedMessage := stripansi.Strip(*failedStatus.Message)
				markdownTest.Message = &strippedMessage
			}
		}

		testMarkdown := new(strings.Builder)
		if err := parsedTemplate.Execute(testMarkdown, markdownTest); err != nil {
			return false, errors.WithStack(err)
		}

		if oneMB-markdown.Len()-testMarkdown.Len()-len(markdownResultsTruncated) <= 0 {
			return true, nil
		}

		if _, err := markdown.WriteString(testMarkdown.String()); err != nil {
			return false, errors.WithStack(err)
		}
	}

	return false, nil
}

func retryTemplateAndSubstitutionFor(
	framework v1.Framework,
	retryCommandTemplate string,
) (*templating.CompiledTemplate, targetedretries.Substitution) {
	// note we don't propagate any errors up here; if we're unable to generate
	// the retry command, we should still provide the rest of the summary

	substitution, ok := targetedretries.SubstitutionsByFramework[framework]
	if !ok {
		return nil, nil
	}

	var exampleTemplate *templating.CompiledTemplate
	if t, err := templating.CompileTemplate(substitution.Example()); err == nil {
		exampleTemplate = &t
	}

	var providedTemplate *templating.CompiledTemplate
	if retryCommandTemplate != "" {
		if t, err := templating.CompileTemplate(retryCommandTemplate); err == nil {
			providedTemplate = &t
		}
	}

	if exampleTemplate != nil {
		if err := substitution.ValidateTemplate(*exampleTemplate); err != nil {
			exampleTemplate = nil
		}
	}

	if providedTemplate != nil {
		if err := substitution.ValidateTemplate(*providedTemplate); err != nil {
			providedTemplate = nil
		}
	}

	if providedTemplate != nil {
		return providedTemplate, substitution
	}

	if exampleTemplate != nil {
		return exampleTemplate, substitution
	}

	return nil, nil
}

func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
