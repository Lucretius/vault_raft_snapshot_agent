package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type LocalConfig struct {
	Path  string `validate:"required_if=Empty false,omitempty,dir"`
	Empty bool
}

type localUploader struct {
	path string
}

func newLocalUploader(config LocalConfig) (*localUploader, error) {
	return &localUploader{
		config.Path,
	}, nil
}

func (u *localUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64, retain int) error {
	fileName := fmt.Sprintf("%s/raft_snapshot-%d.snap", u.path, currentTs)
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating local file: %w", err)
	}
	_, err = io.Copy(file, reader)

	if err != nil {
		return fmt.Errorf("error writing snapshot to local storage: %w", err)
	} else {
		if retain > 0 {
			existingSnapshots, err := u.listUploadedSnapshotsAscending("raft_snapshot-")

			if err != nil {
				return fmt.Errorf("error getting existing snapshots from local storage: %w", err)
			}

			if len(existingSnapshots) <= int(retain) {
				return nil
			}

			filesToDelete := existingSnapshots[0 : len(existingSnapshots)-int(retain)]

			for _, f := range filesToDelete {
				err := os.Remove(fmt.Sprintf("%s/%s", u.path, f.Name()))
				if err != nil {
					return fmt.Errorf("error deleting local snapshot %s: %w", f.Name(), err)
				}
			}
		}
		return nil
	}
}

func (u *localUploader) listUploadedSnapshotsAscending(keyPrefix string) ([]os.FileInfo, error) {
	var result []os.FileInfo

	files, err := os.ReadDir(u.path)

	if err != nil {
		return result, fmt.Errorf("error reading local directory: %w", err)
	}

	for _, file := range files {
		if strings.Contains(file.Name(), keyPrefix) && strings.HasSuffix(file.Name(), ".snap") {
			info, err := file.Info()
			if err != nil {
				return result, fmt.Errorf("error getting local file info: %w", err)
			}
			result = append(result, info)
		}
	}

	timestamp := func(f1, f2 *os.FileInfo) bool {
		file1 := *f1
		file2 := *f2
		return file1.ModTime().Before(file2.ModTime())
	}

	localBy(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for fileInfo
type localBy func(f1, f2 *os.FileInfo) bool

func (by localBy) Sort(files []os.FileInfo) {
	fs := &fileSorter{
		files: files,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type fileSorter struct {
	files []os.FileInfo
	by    localBy
}

func (s *fileSorter) Len() int {
	return len(s.files)
}

func (s *fileSorter) Less(i, j int) bool {
	return s.by(&s.files[i], &s.files[j])
}

func (s *fileSorter) Swap(i, j int) {
	s.files[i], s.files[j] = s.files[j], s.files[i]
}
