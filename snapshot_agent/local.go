package snapshot_agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
)

type LocalUploader struct {
	config config.LocalConfig
	retain int64
}

func NewLocalUploader(config *config.Configuration) (*LocalUploader, error) {
	return &LocalUploader{
		config: config.Local,
		retain: config.Retain,
	}, nil
}

func (u *LocalUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64) (string, error) {
	fileName := fmt.Sprintf("%s/raft_snapshot-%d.snap", u.config.Path, currentTs)
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("error creating file: %w", err)
	}
	_, err = io.Copy(file, reader)

	if err != nil {
		return "", fmt.Errorf("error writing snapshot to local storage: %w", err)
	} else {
		if u.retain > 0 {
			existingSnapshots, err := u.listUploadedSnapshotsAscending("raft_snapshot-")

			if err != nil {
				return "", fmt.Errorf("error getting existing snapshots: %w", err)
			}

			if len(existingSnapshots) <= int(u.retain) {
				return fileName, nil
			}

			filesToDelete := existingSnapshots[0 : len(existingSnapshots)-int(u.retain)]

			for _, f := range filesToDelete {
				err := os.Remove(fmt.Sprintf("%s/%s", u.config.Path, f.Name()))
				if err != nil {
					return "", fmt.Errorf("error deleting snapshot %s: %w", f.Name(), err)
				}
			}
		}
		return fileName, nil
	}
}

func (u *LocalUploader) LastSuccessfulUpload(ctx context.Context) (time.Time, error) {
	existingSnapshots, err := u.listUploadedSnapshotsAscending("raft_snapshot-")

	if err != nil {
		return time.Time{}, fmt.Errorf("error getting existing snapshots: %w", err)
	}

	if len(existingSnapshots) == 0 {
		return time.Time{}, nil
	}

	lastSnapshot := existingSnapshots[len(existingSnapshots)-1]

	return lastSnapshot.ModTime(), nil
}

func (u *LocalUploader) listUploadedSnapshotsAscending(keyPrefix string) ([]os.FileInfo, error) {

	var result []os.FileInfo

	files, err := os.ReadDir(u.config.Path)

	if err != nil {
		return result, fmt.Errorf("error reading directory: %w", err)
	}

	for _, file := range files {
		if strings.Contains(file.Name(), keyPrefix) && strings.HasSuffix(file.Name(), ".snap") {
			info, err := file.Info()
			if err != nil {
				return result, fmt.Errorf("error getting file info: %w", err)
			}
			result = append(result, info)
		}
	}

	timestamp := func(f1, f2 *os.FileInfo) bool {
		file1 := *f1
		file2 := *f2
		return file1.ModTime().Before(file2.ModTime())
	}

	By(timestamp).Sort(result)

	return result, nil
}

// implementation of Sort interface for fileInfo
type By func(f1, f2 *os.FileInfo) bool

func (by By) Sort(files []os.FileInfo) {
	fs := &fileSorter{
		files: files,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type fileSorter struct {
	files []os.FileInfo
	by    func(f1, f2 *os.FileInfo) bool // Closure used in the Less method.
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
