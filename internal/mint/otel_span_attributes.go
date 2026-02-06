package mint

import (
	"encoding/json"
	stdErrors "errors"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

const rwxSuiteIDAttributeKey = "rwx.tests.suite-id"

func WriteOtelSpanAttributesJSON(fileSystem fs.FileSystem, log *zap.SugaredLogger, attrs map[string]any) error {
	otelSpanPath := os.Getenv("RWX_OTEL_SPAN")
	if otelSpanPath == "" {
		return nil
	}

	suiteIDPath := filepath.Join(otelSpanPath, rwxSuiteIDAttributeKey+".json")
	if _, hasSuiteID := attrs[rwxSuiteIDAttributeKey]; hasSuiteID {
		_, err := fileSystem.Stat(suiteIDPath)
		switch {
		case err == nil:
			if log != nil {
				log.Warnf("Skipping RWX OTEL span attributes because they were already written: %s", suiteIDPath)
			}
			return nil
		case !errors.Is(err, os.ErrNotExist):
			return errors.Wrapf(err, "unable to stat RWX OTEL span suite-id attribute path %q", suiteIDPath)
		}
	}

	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var allErrs []error

	for _, key := range keys {
		outputPath := filepath.Join(otelSpanPath, key+".json")
		encodedValue, err := json.Marshal(attrs[key])
		if err != nil {
			allErrs = append(allErrs, errors.Wrapf(err, "unable to JSON encode RWX OTEL span attribute %q", key))
			continue
		}

		file, err := fileSystem.Create(outputPath)
		if err != nil {
			allErrs = append(allErrs, errors.Wrapf(err, "unable to create RWX OTEL span attribute file %q", outputPath))
			continue
		}

		_, writeErr := file.Write(encodedValue)
		closeErr := file.Close()
		if writeErr != nil {
			allErrs = append(allErrs, errors.Wrapf(writeErr, "unable to write RWX OTEL span attribute file %q", outputPath))
			continue
		}
		if closeErr != nil {
			allErrs = append(allErrs, errors.Wrapf(closeErr, "unable to close RWX OTEL span attribute file %q", outputPath))
			continue
		}
	}

	if len(allErrs) == 0 {
		return nil
	}

	return errors.Wrap(
		stdErrors.Join(allErrs...),
		"unable to write one or more RWX OTEL span attributes",
	)
}
