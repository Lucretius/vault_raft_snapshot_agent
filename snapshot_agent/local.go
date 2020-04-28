package snapshot_agent

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
)

// CreateLocalSnapshot writes snapshot to disk location
func (s *Snapshotter) CreateLocalSnapshot(buf *bytes.Buffer, config *config.Configuration, currentTs int64) (string, error) {
	err := ioutil.WriteFile(fmt.Sprintf("%s/raft_snapshot-%d.snap", config.Local.Path, currentTs), buf.Bytes(), 0644)
	if err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%s/raft_snapshot-%d.snap", config.Local.Path, currentTs), nil
	}
}
