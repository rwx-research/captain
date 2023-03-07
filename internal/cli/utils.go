package cli

import (
	"fmt"
	"io"
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

		if err := ias.moveFile(artifact, filepath.Join(targetPath, filename)); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ias *intermediateArtifactStorage) moveFile(srcPath, dstPath string) error {
	// Renaming only works if both paths are on the same file-system. We'll fall back to
	// copy & delete instead if this is not the case.
	if err := ias.fs.Rename(srcPath, dstPath); err == nil {
		return nil
	}

	srcFile, err := ias.fs.Open(srcPath)
	if err != nil {
		return errors.WithStack(err)
	}

	dstFile, err := ias.fs.Create(dstPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = dstFile.Close()
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		_ = srcFile.Close()
		return errors.WithStack(err)
	}

	if err = dstFile.Sync(); err != nil {
		_ = srcFile.Close()
		return errors.WithStack(err)
	}

	if err = srcFile.Close(); err != nil {
		return errors.WithStack(err)
	}

	if err = ias.fs.Remove(srcPath); err != nil {
		return errors.WithStack(err)
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
