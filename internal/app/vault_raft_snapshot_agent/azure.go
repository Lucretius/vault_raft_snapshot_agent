package vault_raft_snapshot_agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

// CreateAzureSnapshot writes snapshot to azure blob storage
func (s *Snapshotter) CreateAzureSnapshot(reader io.ReadWriter, config *Configuration, currentTs int64) (string, error) {
	ctx := context.Background()
	url := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	blob := s.AzureUploader.NewBlockBlobURL(url)
	_, err := azblob.UploadStreamToBlockBlob(ctx, reader, blob, azblob.UploadStreamToBlockBlobOptions{
		BufferSize: 4 * 1024 * 1024,
		MaxBuffers: 16,
	})
	if err != nil {
		return "", err
	} else {
		if config.Retain > 0 {
			deleteCtx := context.Background()
			res, err := s.AzureUploader.ListBlobsFlatSegment(deleteCtx, azblob.Marker{}, azblob.ListBlobsSegmentOptions{
				Prefix:     "raft_snapshot-",
				MaxResults: 500,
			})
			if err != nil {
				log.Println("Unable to iterate through bucket to find old snapshots to delete")
				return url, err
			}
			blobs := res.Segment.BlobItems
			timestamp := func(o1, o2 *azblob.BlobItem) bool {
				return o1.Properties.LastModified.Before(o2.Properties.LastModified)
			}
			AzureBy(timestamp).Sort(blobs)
			if len(blobs)-int(config.Retain) <= 0 {
				return url, nil
			}
			blobsToDelete := blobs[0 : len(blobs)-int(config.Retain)]

			for _, b := range blobsToDelete {
				val := s.AzureUploader.NewBlockBlobURL(b.Name)
				_, err := val.Delete(deleteCtx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
				if err != nil {
					log.Println("Cannot delete old snapshot")
					return url, err
				}
			}
		}
		return url, nil
	}
}

// implementation of Sort interface for s3 objects
type AzureBy func(f1, f2 *azblob.BlobItem) bool

func (by AzureBy) Sort(objects []azblob.BlobItem) {
	fs := &azObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type azObjectSorter struct {
	objects []azblob.BlobItem
	by      func(f1, f2 *azblob.BlobItem) bool // Closure used in the Less method.
}

func (s *azObjectSorter) Len() int {
	return len(s.objects)
}

func (s *azObjectSorter) Less(i, j int) bool {
	return s.by(&s.objects[i], &s.objects[j])
}

func (s *azObjectSorter) Swap(i, j int) {
	s.objects[i], s.objects[j] = s.objects[j], s.objects[i]
}
