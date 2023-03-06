package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

type intermediateArtifactStorage struct {
	basePath   string
	commandID  string
	fs         fs.FileSystem
	retryID    string
	workingDir string
}

func (s Service) newIntermediateArtifactStorage(path string) (*intermediateArtifactStorage, error) {
	wd, err := s.FileSystem.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ias := &intermediateArtifactStorage{
		basePath:   path,
		fs:         s.FileSystem,
		workingDir: wd,
		retryID:    "original-attempt",
	}

	if path == "" {
		path, err := s.FileSystem.MkdirTemp("", "captain")
		if err != nil {
			return nil, errors.WithStack(err)
		}

		ias.basePath = path

		return ias, nil
	}

	info, err := s.FileSystem.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		err = s.FileSystem.MkdirAll(path, 0o750)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if info != nil && !info.IsDir() {
		return nil, errors.NewConfigurationError("intermediate artifacts path %q needs to be a directory", path)
	}

	return ias, nil
}

func (ias *intermediateArtifactStorage) moveTestResults(artifacts []string) error {
	var err error

	attemptPath := filepath.Join(ias.basePath, ias.retryID)
	if ias.commandID != "" {
		attemptPath = filepath.Join(attemptPath, ias.commandID)
	}

	for _, artifact := range artifacts {
		dir, filename := filepath.Split(artifact)

		if filepath.IsAbs(dir) {
			dir, err = filepath.Rel(ias.workingDir, dir)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		if dir != "" && !fs.IsLocal(dir) {
			return errors.NewConfigurationError("test results are outside of working directory")
		}

		targetPath := filepath.Join(attemptPath, dir)
		if err := ias.fs.MkdirAll(targetPath, 0o750); err != nil {
			return errors.WithStack(err)
		}

		newPath := filepath.Join(targetPath, filename)
		if err := ias.fs.Rename(artifact, newPath); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ias *intermediateArtifactStorage) delete() error {
	return errors.WithStack(ias.fs.RemoveAll(ias.basePath))
}

func (ias *intermediateArtifactStorage) setCommandID(n int) {
	ias.commandID = fmt.Sprintf("command-%d", n)
}

func (ias *intermediateArtifactStorage) setRetryID(n int) {
	ias.retryID = fmt.Sprintf("retry-%d", n)
}
