package snapshot_agent

import (
	"context"
	"fmt"
	"io"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
)

// CreateGCPSnapshot writes snapshot to google storage
func (s *Snapshotter) CreateGCPSnapshot(reader io.ReadWriter, config *config.Configuration, currentTs int64) (string, error) {
	obj := s.GCPBucket.Object(fmt.Sprintf("raft_snapshot-%d.snap", currentTs))
	w := obj.NewWriter(context.Background())
	if _, err := io.Copy(w, reader); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("raft_snapshot-%d.snap", currentTs), nil
	}
}
