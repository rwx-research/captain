package cli_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCli(t *testing.T) {
	t.Parallel()

	// Remove any possible inherited ABQ environment variables to test overrides.
	os.Unsetenv("ABQ_SET_EXIT_CODE")
	os.Unsetenv("ABQ_STATE_FILE")

	RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}
