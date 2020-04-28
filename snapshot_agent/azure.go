package snapshot_agent

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Lucretius/vault_raft_snapshot_agent/config"
)

// CreateAzureSnapshot writes snapshot to azure blob storage
func (s *Snapshotter) CreateAzureSnapshot(reader io.ReadWriter, config *config.Configuration, currentTs int64) (string, error) {
	ctx := context.Background()
	url := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	blob := s.AzureUploader.NewBlockBlobURL("test")
	_, err := azblob.UploadStreamToBlockBlob(ctx, reader, blob, azblob.UploadStreamToBlockBlobOptions{
		BufferSize: 4 * 1024 * 1024,
		MaxBuffers: 16,
	})
	if err != nil {
		return "", err
	} else {
		return url, nil
	}

}
