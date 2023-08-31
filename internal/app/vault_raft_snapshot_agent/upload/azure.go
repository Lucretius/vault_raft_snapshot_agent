package upload

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type AzureConfig struct {
	AccountName   string `validate:"required_if=Empty false"`
	AccountKey    string `validate:"required_if=Empty false"`
	ContainerName string `mapstructure:"container" validate:"required_if=Empty false"`
	Empty bool
}

type azureUploader struct {
	azureUploader *azblob.Client
	containerName string
}

func newAzureUploader(config AzureConfig) (*azureUploader, error) {
	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials for azure: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.AccountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}

	return &azureUploader{
		client,
		config.ContainerName,
	}, nil
}

func (u *azureUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64, retain int) error {
	name := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)
	_, err := u.azureUploader.UploadStream(ctx, u.containerName, name, reader, &azblob.UploadStreamOptions{
		BlockSize:   4 * 1024 * 1024,
		Concurrency: 16,
	})

	if err != nil {
		return fmt.Errorf("error uploading snapshot to azure: %w", err)
	}

	if retain > 0 {
		existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

		if err != nil {
			return fmt.Errorf("error getting existing snapshots from azure: %w", err)
		}

		if len(existingSnapshots)-int(retain) <= 0 {
			return nil
		}

		blobsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(retain)]

		for _, b := range blobsToDelete {
			_, err := u.azureUploader.DeleteBlob(ctx, u.containerName, *b.Name, nil)
			if err != nil {
				return fmt.Errorf("error deleting snapshot %s from azure: %w", *b.Name, err)
			}
		}
	}
	return nil
}

func (u *azureUploader) listUploadedSnapshotsAscending(ctx context.Context, keyPrefix string) ([]*container.BlobItem, error) {
	var results []*container.BlobItem

	var maxResults int32 = 500

	pager := u.azureUploader.NewListBlobsFlatPager(u.containerName, &azblob.ListBlobsFlatOptions{
		Prefix:     &keyPrefix,
		MaxResults: &maxResults,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)

		if err != nil {
			return results, fmt.Errorf("error paging blobs on azure: %w", err)
		}

		results = append(results, resp.Segment.BlobItems...)
	}

	timestamp := func(o1, o2 *container.BlobItem) bool {
		return o1.Properties.LastModified.Before(*o2.Properties.LastModified)
	}

	azureBy(timestamp).Sort(results)

	return results, nil
}

// implementation of Sort interface for s3 objects
type azureBy func(f1, f2 *container.BlobItem) bool

func (by azureBy) Sort(objects []*container.BlobItem) {
	fs := &azObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type azObjectSorter struct {
	objects []*container.BlobItem
	by      azureBy
}

func (s *azObjectSorter) Len() int {
	return len(s.objects)
}

func (s *azObjectSorter) Less(i, j int) bool {
	return s.by(s.objects[i], s.objects[j])
}

func (s *azObjectSorter) Swap(i, j int) {
	s.objects[i], s.objects[j] = s.objects[j], s.objects[i]
}
