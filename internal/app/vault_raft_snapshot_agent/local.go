package vault_raft_snapshot_agent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

// CreateLocalSnapshot writes snapshot to disk location
func (s *Snapshotter) CreateLocalSnapshot(buf *bytes.Buffer, config *Configuration, currentTs int64) (string, error) {
	fileName := fmt.Sprintf("%s/raft_snapshot-%d.snap", config.Local.Path, currentTs)
	err := ioutil.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		return "", err
	} else {
		if config.Retain > 0 {
			fileInfo, err := ioutil.ReadDir(config.Local.Path)
			filesToDelete := make([]os.FileInfo, 0)
			for _, file := range fileInfo {
				if strings.Contains(file.Name(), "raft_snapshot-") && strings.HasSuffix(file.Name(), ".snap") {
					filesToDelete = append(filesToDelete, file)
				}
			}
			if err != nil {
				log.Println("Unable to read file directory to delete old snapshots")
				return fileName, err
			}
			timestamp := func(f1, f2 *os.FileInfo) bool {
				file1 := *f1
				file2 := *f2
				return file1.ModTime().Before(file2.ModTime())
			}
			By(timestamp).Sort(filesToDelete)
			if len(filesToDelete) <= int(config.Retain) {
				return fileName, nil
			}
			filesToDelete = filesToDelete[0 : len(filesToDelete)-int(config.Retain)]
			for _, f := range filesToDelete {
				os.Remove(fmt.Sprintf("%s/%s", config.Local.Path, f.Name()))
			}
		}
		return fileName, nil
	}
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
