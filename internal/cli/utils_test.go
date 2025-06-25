package cli_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/rwx-research/captain-cli/internal/cli"
	"github.com/rwx-research/captain-cli/internal/fs"
	"github.com/rwx-research/captain-cli/internal/mocks"
)

var (
	errCrossDeviceLink = errors.New("cross-device link")
	errGlob            = errors.New("glob error")
	errMkdir           = errors.New("mkdir error")
	errOpen            = errors.New("open error")
	errGetwd           = errors.New("getwd error")
)

var _ = ginkgo.Describe("IntermediateArtifactStorage", func() {
	var (
		service    cli.Service
		mockFS     *mocks.FileSystem
		workingDir string
		basePath   string
		ias        *cli.IntermediateArtifactStorage
	)

	ginkgo.BeforeEach(func() {
		workingDir = "/working/dir"
		basePath = fmt.Sprintf("intermediate-results-%d", ginkgo.GinkgoRandomSeed())

		mockFS = new(mocks.FileSystem)
		service = cli.Service{
			FileSystem: mockFS,
		}

		mockFS.MockGetwd = func() (string, error) {
			return workingDir, nil
		}

		mockFS.MockStat = func(name string) (os.FileInfo, error) {
			if name == basePath {
				return nil, os.ErrNotExist
			}
			return nil, os.ErrNotExist
		}

		mockFS.MockMkdirAll = func(_ string, _ os.FileMode) error {
			return nil
		}

		var err error
		ias, err = service.NewIntermediateArtifactStorage(basePath)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Describe("MoveAdditionalArtifacts", func() {
		ginkgo.Context("when no artifact patterns are provided", func() {
			ginkgo.It("returns nil without calling filesystem operations", func() {
				err := ias.MoveAdditionalArtifacts([]string{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("when artifact patterns are provided", func() {
			var (
				patterns  []string
				artifacts []string
			)

			ginkgo.BeforeEach(func() {
				patterns = []string{"coverage/**/*", "reports/**/*", "/tmp/external/**/*"}
				artifacts = []string{
					"coverage/lcov.info",
					"reports/junit.xml",
					"/tmp/external/debug.log",
					"/working/dir/subdir/artifact.txt",
				}

				mockFS.MockGlobMany = func(p []string) ([]string, error) {
					gomega.Expect(p).To(gomega.Equal(patterns))
					return artifacts, nil
				}

				mockFS.MockRename = func(_, _ string) error {
					return nil
				}
			})

			ginkgo.It("moves artifacts with correct path transformations", func() {
				var renamedPaths [][]string
				mockFS.MockRename = func(oldpath, newpath string) error {
					renamedPaths = append(renamedPaths, []string{oldpath, newpath})
					return nil
				}

				err := ias.MoveAdditionalArtifacts(patterns)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(renamedPaths).To(gomega.HaveLen(4))

				// Relative path: coverage/lcov.info -> __captain_working_directory/coverage/lcov.info
				gomega.Expect(renamedPaths[0][0]).To(gomega.Equal("coverage/lcov.info"))
				expectedPath0 := filepath.Join(basePath, "original-attempt", "__captain_working_directory", "coverage", "lcov.info")
				gomega.Expect(renamedPaths[0][1]).To(gomega.Equal(expectedPath0))

				// Relative path: reports/junit.xml -> __captain_working_directory/reports/junit.xml
				gomega.Expect(renamedPaths[1][0]).To(gomega.Equal("reports/junit.xml"))
				expectedPath1 := filepath.Join(basePath, "original-attempt", "__captain_working_directory", "reports", "junit.xml")
				gomega.Expect(renamedPaths[1][1]).To(gomega.Equal(expectedPath1))

				// Absolute path outside working directory: /tmp/external/debug.log -> tmp/external/debug.log
				gomega.Expect(renamedPaths[2][0]).To(gomega.Equal("/tmp/external/debug.log"))
				expectedPath2 := filepath.Join(basePath, "original-attempt", "tmp", "external", "debug.log")
				gomega.Expect(renamedPaths[2][1]).To(gomega.Equal(expectedPath2))

				// Absolute path inside working directory: /working/dir/subdir/artifact.txt -> __captain_working_directory/subdir/artifact.txt
				gomega.Expect(renamedPaths[3][0]).To(gomega.Equal("/working/dir/subdir/artifact.txt"))
				expectedPath3 := filepath.Join(basePath, "original-attempt",
					"__captain_working_directory", "subdir", "artifact.txt")
				gomega.Expect(renamedPaths[3][1]).To(gomega.Equal(expectedPath3))
			})

			ginkgo.It("creates necessary directories before moving files", func() {
				var createdDirs []string
				mockFS.MockMkdirAll = func(path string, _ os.FileMode) error {
					createdDirs = append(createdDirs, path)
					return nil
				}

				err := ias.MoveAdditionalArtifacts(patterns)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(createdDirs).To(gomega.HaveLen(4))
				expectedDir0 := filepath.Join(basePath, "original-attempt", "__captain_working_directory", "coverage")
				gomega.Expect(createdDirs).To(gomega.ContainElement(expectedDir0))
				expectedDir1 := filepath.Join(basePath, "original-attempt", "__captain_working_directory", "reports")
				gomega.Expect(createdDirs).To(gomega.ContainElement(expectedDir1))
				expectedDir2 := filepath.Join(basePath, "original-attempt", "tmp", "external")
				gomega.Expect(createdDirs).To(gomega.ContainElement(expectedDir2))
				expectedDir3 := filepath.Join(basePath, "original-attempt", "__captain_working_directory", "subdir")
				gomega.Expect(createdDirs).To(gomega.ContainElement(expectedDir3))
			})

			ginkgo.It("handles retry attempts by updating the retry ID", func() {
				ias.SetRetryID(2)

				var renamedPaths [][]string
				mockFS.MockRename = func(oldpath, newpath string) error {
					renamedPaths = append(renamedPaths, []string{oldpath, newpath})
					return nil
				}

				err := ias.MoveAdditionalArtifacts(patterns)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(renamedPaths[0][1]).To(gomega.ContainSubstring("retry-2"))
				gomega.Expect(renamedPaths[0][1]).NotTo(gomega.ContainSubstring("original-attempt"))
			})

			ginkgo.It("handles command ID by including it in the path", func() {
				ias.SetCommandID(3)

				var renamedPaths [][]string
				mockFS.MockRename = func(oldpath, newpath string) error {
					renamedPaths = append(renamedPaths, []string{oldpath, newpath})
					return nil
				}

				err := ias.MoveAdditionalArtifacts(patterns)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(renamedPaths[0][1]).To(gomega.ContainSubstring("command-3"))
			})

			ginkgo.It("falls back to copy+delete when rename fails", func() {
				mockFS.MockRename = func(_, _ string) error {
					return errCrossDeviceLink
				}

				var openedFiles []string
				var createdFiles []string
				var removedFiles []string

				mockFS.MockOpen = func(path string) (fs.File, error) {
					openedFiles = append(openedFiles, path)
					return &mocks.File{
						Builder: new(strings.Builder),
						Reader:  strings.NewReader("test content"),
					}, nil
				}

				mockFS.MockCreate = func(name string) (fs.File, error) {
					createdFiles = append(createdFiles, name)
					return &mocks.File{
						Builder: new(strings.Builder),
						Reader:  strings.NewReader(""),
					}, nil
				}

				mockFS.MockRemove = func(name string) error {
					removedFiles = append(removedFiles, name)
					return nil
				}

				err := ias.MoveAdditionalArtifacts(patterns)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(openedFiles).To(gomega.HaveLen(4))
				gomega.Expect(openedFiles).To(gomega.ContainElement("coverage/lcov.info"))
				gomega.Expect(openedFiles).To(gomega.ContainElement("reports/junit.xml"))
				gomega.Expect(openedFiles).To(gomega.ContainElement("/tmp/external/debug.log"))
				gomega.Expect(openedFiles).To(gomega.ContainElement("/working/dir/subdir/artifact.txt"))

				gomega.Expect(createdFiles).To(gomega.HaveLen(4))

				gomega.Expect(removedFiles).To(gomega.HaveLen(4))
				gomega.Expect(removedFiles).To(gomega.ContainElement("coverage/lcov.info"))
				gomega.Expect(removedFiles).To(gomega.ContainElement("reports/junit.xml"))
				gomega.Expect(removedFiles).To(gomega.ContainElement("/tmp/external/debug.log"))
				gomega.Expect(removedFiles).To(gomega.ContainElement("/working/dir/subdir/artifact.txt"))
			})
		})

		ginkgo.Context("when glob operation fails", func() {
			ginkgo.It("returns the error", func() {
				mockFS.MockGlobMany = func(_ []string) ([]string, error) {
					return nil, errGlob
				}

				err := ias.MoveAdditionalArtifacts([]string{"coverage/**/*"})
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("glob error"))
			})
		})

		ginkgo.Context("when directory creation fails", func() {
			ginkgo.It("returns the error", func() {
				mockFS.MockGlobMany = func(_ []string) ([]string, error) {
					return []string{"coverage/lcov.info"}, nil
				}

				mockFS.MockMkdirAll = func(_ string, _ os.FileMode) error {
					return errMkdir
				}

				err := ias.MoveAdditionalArtifacts([]string{"coverage/**/*"})
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("mkdir error"))
			})
		})

		ginkgo.Context("when file move operation fails completely", func() {
			ginkgo.It("returns the error", func() {
				mockFS.MockGlobMany = func(_ []string) ([]string, error) {
					return []string{"coverage/lcov.info"}, nil
				}

				mockFS.MockMkdirAll = func(_ string, _ os.FileMode) error {
					return nil
				}

				mockFS.MockRename = func(_, _ string) error {
					return errCrossDeviceLink
				}

				mockFS.MockOpen = func(_ string) (fs.File, error) {
					return nil, errOpen
				}

				err := ias.MoveAdditionalArtifacts([]string{"coverage/**/*"})
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("open error"))
			})
		})
	})

	ginkgo.Describe("NewIntermediateArtifactStorage", func() {
		ginkgo.Context("when path is provided", func() {
			ginkgo.It("creates storage with the provided path", func() {
				providedPath := "/custom/artifacts/path"

				mockFS.MockStat = func(name string) (os.FileInfo, error) {
					gomega.Expect(name).To(gomega.Equal(providedPath))
					return nil, os.ErrNotExist
				}

				mockFS.MockMkdirAll = func(path string, _ os.FileMode) error {
					gomega.Expect(path).To(gomega.Equal(providedPath))
					return nil
				}

				storage, err := service.NewIntermediateArtifactStorage(providedPath)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(storage).NotTo(gomega.BeNil())
			})

			ginkgo.It("validates that the path is not a file", func() {
				providedPath := "/custom/artifacts/path"

				mockFileInfo := mocks.FileInfo{Dir: false}

				mockFS.MockStat = func(_ string) (os.FileInfo, error) {
					return mockFileInfo, nil
				}

				_, err := service.NewIntermediateArtifactStorage(providedPath)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("Intermediate artifacts path is not a directory"))
			})
		})

		ginkgo.Context("when no path is provided", func() {
			ginkgo.It("creates a temporary directory", func() {
				tempDir := "/tmp/captain123"

				mockFS.MockMkdirTemp = func(_ string, pattern string) (string, error) {
					gomega.Expect(pattern).To(gomega.Equal("captain"))
					return tempDir, nil
				}

				storage, err := service.NewIntermediateArtifactStorage("")
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(storage).NotTo(gomega.BeNil())
			})
		})

		ginkgo.Context("when working directory cannot be determined", func() {
			ginkgo.It("returns an error", func() {
				mockFS.MockGetwd = func() (string, error) {
					return "", errGetwd
				}

				_, err := service.NewIntermediateArtifactStorage(basePath)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("getwd error"))
			})
		})
	})
})
