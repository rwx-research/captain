package mint_test

import (
	stdErrors "errors"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mint"
	"github.com/rwx-research/captain-cli/internal/mocks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var errFailedToCreateFile = stdErrors.New("failed to create file")

var _ = Describe("WriteOtelSpanAttributesJSON", func() {
	It("is a no-op when RWX_OTEL_SPAN is unset", func() {
		Expect(os.Unsetenv("RWX_OTEL_SPAN")).To(Succeed())

		logger, _ := testLogger()
		err := mint.WriteOtelSpanAttributesJSON(&mocks.FileSystem{}, logger, map[string]any{
			"rwx.tests.summary.tests": 1,
		})

		Expect(err).NotTo(HaveOccurred())
	})

	It("writes typed JSON values", func() {
		otelSpanDir, err := os.MkdirTemp("", "otel-span-attrs-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(otelSpanDir)).To(Succeed())
		})

		Expect(os.Setenv("RWX_OTEL_SPAN", otelSpanDir)).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("RWX_OTEL_SPAN")).To(Succeed())
		})

		logger, _ := testLogger()
		err = mint.WriteOtelSpanAttributesJSON(fs.Local{}, logger, map[string]any{
			"rwx.tests.suite-id":       "suite-1",
			"rwx.tests.summary.tests":  12,
			"rwx.tests.summary.flaky":  2,
			"rwx.tests.summary.status": "failed",
			"rwx.tests.retry.enabled":  true,
			"rwx.tests.retry.note":     nil,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.suite-id.json"))).To(Equal(`"suite-1"`))
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.summary.tests.json"))).To(Equal(`12`))
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.summary.flaky.json"))).To(Equal(`2`))
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.summary.status.json"))).To(Equal(`"failed"`))
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.retry.enabled.json"))).To(Equal(`true`))
		Expect(readFile(filepath.Join(otelSpanDir, "rwx.tests.retry.note.json"))).To(Equal(`null`))
	})

	It("skips all attributes when the suite-id attribute already exists", func() {
		otelSpanDir, err := os.MkdirTemp("", "otel-span-attrs-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(otelSpanDir)).To(Succeed())
		})

		Expect(os.Setenv("RWX_OTEL_SPAN", otelSpanDir)).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("RWX_OTEL_SPAN")).To(Succeed())
		})

		existingPath := filepath.Join(otelSpanDir, "rwx.tests.suite-id.json")
		Expect(os.WriteFile(existingPath, []byte(`99`), 0o600)).To(Succeed())

		logger, recordedLogs := testLogger()
		err = mint.WriteOtelSpanAttributesJSON(fs.Local{}, logger, map[string]any{
			"rwx.tests.suite-id":      "suite-1",
			"rwx.tests.summary.tests": 7,
			"rwx.tests.summary.flaky": 1,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(readFile(existingPath)).To(Equal(`99`))
		_, err = os.Stat(filepath.Join(otelSpanDir, "rwx.tests.summary.tests.json"))
		Expect(os.IsNotExist(err)).To(BeTrue())
		_, err = os.Stat(filepath.Join(otelSpanDir, "rwx.tests.summary.flaky.json"))
		Expect(os.IsNotExist(err)).To(BeTrue())

		sawWarning := false
		for _, entry := range recordedLogs.All() {
			if strings.Contains(entry.Message, "already written") && strings.Contains(entry.Message, existingPath) {
				sawWarning = true
				break
			}
		}

		Expect(sawWarning).To(BeTrue())
	})

	It("returns an error when creating an attribute file fails", func() {
		Expect(os.Setenv("RWX_OTEL_SPAN", "/tmp/rwx-otel-span")).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("RWX_OTEL_SPAN")).To(Succeed())
		})

		logger, _ := testLogger()
		err := mint.WriteOtelSpanAttributesJSON(&mocks.FileSystem{
			MockStat: func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MockCreate: func(string) (fs.File, error) {
				return nil, errFailedToCreateFile
			},
		}, logger, map[string]any{
			"rwx.tests.summary.tests": 2,
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to write one or more RWX OTEL span attributes"))
		Expect(err.Error()).To(ContainSubstring("unable to create RWX OTEL span attribute file"))
	})

	It("returns an error when marshalling fails", func() {
		Expect(os.Setenv("RWX_OTEL_SPAN", "/tmp/rwx-otel-span")).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("RWX_OTEL_SPAN")).To(Succeed())
		})

		logger, _ := testLogger()
		err := mint.WriteOtelSpanAttributesJSON(&mocks.FileSystem{
			MockStat: func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
		}, logger, map[string]any{
			"rwx.tests.summary.tests": func() {},
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to JSON encode RWX OTEL span attribute"))
	})
})

func readFile(path string) string {
	data, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

func testLogger() (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, recordedLogs := observer.New(zapcore.InfoLevel)
	return zap.New(core).Sugar(), recordedLogs
}
