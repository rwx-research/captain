package targetedretries

import (
	"github.com/rwx-research/captain-cli/internal/templating"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type Substitution interface {
	Example() string
	ValidateTemplate(compiledTemplate templating.CompiledTemplate) error
	SubstitutionsFor(
		_ templating.CompiledTemplate,
		testResults v1.TestResults,
		filter func(v1.Test) bool,
	) ([]map[string]string, error)
}

var SubstitutionsByFramework = map[v1.Framework]Substitution{
	v1.DotNetxUnitFramework:          new(DotNetxUnitSubstitution),
	v1.ElixirExUnitFramework:         new(ElixirExUnitSubstitution),
	v1.GoGinkgoFramework:             new(GoGinkgoSubstitution),
	v1.GoTestFramework:               new(GoTestSubstitution),
	v1.JavaScriptCucumberFramework:   new(JavaScriptCucumberSubstitution),
	v1.JavaScriptCypressFramework:    new(JavaScriptCypressSubstitution),
	v1.JavaScriptJestFramework:       new(JavaScriptJestSubstitution),
	v1.JavaScriptMochaFramework:      new(JavaScriptMochaSubstitution),
	v1.JavaScriptPlaywrightFramework: new(JavaScriptPlaywrightSubstitution),
	v1.JavaScriptVitestFramework:     new(JavaScriptVitestSubstitution),
	v1.JavaScriptBunFramework:        new(JavaScriptBunSubstitution),
	v1.PHPUnitFramework:              new(PHPUnitSubstitution),
	v1.PythonPytestFramework:         new(PythonPytestSubstitution),
	v1.PythonUnitTestFramework:       new(PythonUnitTestSubstitution),
	v1.RubyCucumberFramework:         new(RubyCucumberSubstitution),
	v1.RubyMinitestFramework:         new(RubyMinitestSubstitution),
	v1.RubyRSpecFramework:            new(RubyRSpecSubstitution),
}
