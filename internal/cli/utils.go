package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwx-research/captain-cli/internal/errors"
	"github.com/rwx-research/captain-cli/internal/fs"
)

type IntermediateArtifactStorage struct {
	basePath   string
	commandID  string
	fs         fs.FileSystem
	retryID    string
	workingDir string
}

func (s Service) NewIntermediateArtifactStorage(path string) (*IntermediateArtifactStorage, error) {
	wd, err := s.FileSystem.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ias := &IntermediateArtifactStorage{
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
		return nil, errors.NewConfigurationError(
			"Intermediate artifacts path is not a directory",
			fmt.Sprintf(
				"You specified %q as the path for intermediate artifacts. However, this appears to be a file, not a "+
					"directory.",
				path,
			),
			"Please make sure that the intermediate artifacts path points to new path or an existing directory. Captain "+
				"will create a directory in case the path doesn't exist yet.",
		)
	}

	return ias, nil
}

func (ias *IntermediateArtifactStorage) moveTestResults(artifacts []string) error {
	attemptPath := filepath.Join(ias.basePath, ias.retryID)
	if ias.commandID != "" {
		attemptPath = filepath.Join(attemptPath, ias.commandID)
	}

	for _, artifact := range artifacts {
		dir, filename := filepath.Split(artifact)

		var targetDir string
		if filepath.IsAbs(dir) {
			// For absolute paths outside working directory, preserve full structure
			relDir, err := filepath.Rel(ias.workingDir, dir)
			if err != nil || !fs.IsLocal(relDir) {
				// Path is outside working directory, use full absolute path
				targetDir = filepath.Join(attemptPath, strings.TrimPrefix(dir, "/"))
			} else {
				// Path is inside working directory, use __captain_working_directory prefix
				targetDir = filepath.Join(attemptPath, "__captain_working_directory", relDir)
			}
		} else {
			// Relative paths go under __captain_working_directory
			targetDir = filepath.Join(attemptPath, "__captain_working_directory", dir)
		}

		if err := ias.fs.MkdirAll(targetDir, 0o750); err != nil {
			return errors.WithStack(err)
		}

		if err := ias.moveFile(artifact, filepath.Join(targetDir, filename)); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ias *IntermediateArtifactStorage) MoveAdditionalArtifacts(artifactPatterns []string) error {
	if len(artifactPatterns) == 0 {
		return nil
	}

	// Find all artifacts matching the patterns
	artifacts, err := ias.fs.GlobMany(artifactPatterns)
	if err != nil {
		return errors.WithStack(err)
	}

	attemptPath := filepath.Join(ias.basePath, ias.retryID)
	if ias.commandID != "" {
		attemptPath = filepath.Join(attemptPath, ias.commandID)
	}

	for _, artifact := range artifacts {
		dir, filename := filepath.Split(artifact)

		var targetDir string
		if filepath.IsAbs(dir) {
			// For absolute paths outside working directory, preserve full structure
			relDir, err := filepath.Rel(ias.workingDir, dir)
			if err != nil || !fs.IsLocal(relDir) {
				// Path is outside working directory, use full absolute path
				targetDir = filepath.Join(attemptPath, strings.TrimPrefix(dir, "/"))
			} else {
				// Path is inside working directory, use __captain_working_directory prefix
				targetDir = filepath.Join(attemptPath, "__captain_working_directory", relDir)
			}
		} else {
			// Relative paths go under __captain_working_directory
			targetDir = filepath.Join(attemptPath, "__captain_working_directory", dir)
		}

		if err := ias.fs.MkdirAll(targetDir, 0o750); err != nil {
			return errors.WithStack(err)
		}

		if err := ias.moveFile(artifact, filepath.Join(targetDir, filename)); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ias *IntermediateArtifactStorage) moveFile(srcPath, dstPath string) error {
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

func (ias *IntermediateArtifactStorage) delete() error {
	return errors.WithStack(ias.fs.RemoveAll(ias.basePath))
}

func (ias *IntermediateArtifactStorage) SetCommandID(n int) {
	ias.commandID = fmt.Sprintf("command-%d", n)
}

func (ias *IntermediateArtifactStorage) SetRetryID(n int) {
	ias.retryID = fmt.Sprintf("retry-%d", n)
}
