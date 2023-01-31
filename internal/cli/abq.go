package cli

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/rwx-research/captain-cli/internal/abq"
	"github.com/rwx-research/captain-cli/internal/errors"
)

func (s Service) generateAbqStateFilePath() string {
	randNum, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		panic(err)
	}

	fileName := fmt.Sprintf("captain-abq-%d-%d.json", time.Now().UnixMilli(), randNum)
	return path.Join(s.FileSystem.TempDir(), fileName)
}

func (s Service) setAbqExitCode(ctx context.Context, captainErr error) error {
	stateFilePath := abq.StateFilePath(ctx)
	if len(stateFilePath) == 0 {
		s.Log.Debugln("No ABQ state file was set on context")
		return nil
	}

	file, err := s.FileSystem.Open(stateFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.Log.Debugf("No ABQ state file found at %q", stateFilePath)
			return nil // nolint:nilerr
		}
		return errors.Wrap(err, fmt.Sprintf("Error opening ABQ state file at %q", stateFilePath))
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error reading ABQ state file at %q", stateFilePath))
	}

	var state abq.State
	err = json.Unmarshal(contents, &state)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error decoding ABQ state file at %q", stateFilePath))
	}

	if !state.Supervisor {
		s.Log.Debug("Not ABQ supervisor, so not setting exit code")
		return nil
	}

	var exitCode uint8
	if captainErr == nil {
		exitCode = 0
	} else {
		exitCode = 1
	}

	_, err = s.runCommand(ctx, []string{
		state.AbqExecutable,
		"set-exit-code",
		"--run-id", state.RunID,
		"--exit-code", fmt.Sprint(exitCode),
	}, false)
	if err != nil {
		err = errors.Wrap(err, "Error setting ABQ exit code")
	}

	return err
}

func (s Service) applyAbqEnvironment(ctx context.Context) (context.Context, []string) {
	var environ []string

	const envKey string = "ABQ_STATE_FILE"
	var stateFilePath string
	var ok bool

	// Only set the state file in the environment when not already set.
	if stateFilePath, ok = os.LookupEnv(envKey); !ok {
		stateFilePath = s.generateAbqStateFilePath()
		environ = append(environ, fmt.Sprintf("%s=%s", envKey, stateFilePath))
	}

	// Always force ABQ_SET_EXIT_CODE=false.
	// There is no use case to support allowing a user to override it.
	environ = append(environ, "ABQ_SET_EXIT_CODE=false")

	// Whether state file was pulled from existing environment or generated,
	// always set it on the context.
	ctx = abq.WithStateFilePath(ctx, stateFilePath)
	return ctx, environ
}
