package snapshot_agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Lucretius/vault_raft_snapshot_agent/config"
)

type AzureUploader struct {
	azureUploader *azblob.Client
	config        config.AzureConfig
	retain        int64
}

func NewAzureUploader(config *config.Configuration) (*AzureUploader, error) {
	accountName := config.Azure.AccountName
	if os.Getenv("AZURE_STORAGE_ACCOUNT") != "" {
		accountName = os.Getenv("AZURE_STORAGE_ACCOUNT")
	}
	accountKey := config.Azure.AccountKey
	if os.Getenv("AZURE_STORAGE_ACCESS_KEY") != "" {
		accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	}
	if len(accountName) == 0 || len(accountKey) == 0 {
		return nil, errors.New("invalid Azure configuration")
	}
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials with error: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create Azure client: %w", err)
	}

	return &AzureUploader{
		azureUploader: client,
		config:        config.Azure,
		retain:        config.Retain,
	}, nil
}

func (u *AzureUploader) Upload(ctx context.Context, reader io.Reader, currentTs int64) (string, error) {

	name := fmt.Sprintf("raft_snapshot-%d.snap", currentTs)

	_, err := u.azureUploader.UploadStream(ctx, u.config.ContainerName, name, reader, &azblob.UploadStreamOptions{
		BlockSize:   4 * 1024 * 1024,
		Concurrency: 16,
	})

	if err != nil {
		return "", fmt.Errorf("error uploading snapshot: %w", err)
	} else {
		if u.retain > 0 {
			existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

			if err != nil {
				return "", fmt.Errorf("error getting existing snapshots: %w", err)
			}

			if len(existingSnapshots)-int(u.retain) <= 0 {
				return name, nil
			}

			blobsToDelete := existingSnapshots[0 : len(existingSnapshots)-int(u.retain)]

			for _, b := range blobsToDelete {
				_, err := u.azureUploader.DeleteBlob(ctx, u.config.ContainerName, *b.Name, nil)
				if err != nil {
					return "", fmt.Errorf("error deleting snapshot %s: %w", *b.Name, err)
				}
			}
		}
		return name, nil
	}
}

func (u *AzureUploader) LastSuccessfulUpload(ctx context.Context) (time.Time, error) {

	existingSnapshots, err := u.listUploadedSnapshotsAscending(ctx, "raft_snapshot-")

	if err != nil {
		return time.Time{}, fmt.Errorf("error getting existing snapshots: %w", err)
	}

	if len(existingSnapshots) == 0 {
		return time.Time{}, nil
	}

	lastSnapshot := existingSnapshots[len(existingSnapshots)-1]

	return *lastSnapshot.Properties.LastModified, nil
}

func (u *AzureUploader) listUploadedSnapshotsAscending(ctx context.Context, keyPrefix string) ([]*container.BlobItem, error) {

	var results []*container.BlobItem

	var maxResults int32 = 500

	pager := u.azureUploader.NewListBlobsFlatPager(u.config.ContainerName, &azblob.ListBlobsFlatOptions{
		Prefix:     &keyPrefix,
		MaxResults: &maxResults,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)

		if err != nil {
			return results, fmt.Errorf("error paging blobs: %w", err)
		}

		results = append(results, resp.Segment.BlobItems...)
	}

	timestamp := func(o1, o2 *container.BlobItem) bool {
		return o1.Properties.LastModified.Before(*o2.Properties.LastModified)
	}

	AzureBy(timestamp).Sort(results)

	return results, nil
}

// implementation of Sort interface for s3 objects
type AzureBy func(f1, f2 *container.BlobItem) bool

func (by AzureBy) Sort(objects []*container.BlobItem) {
	fs := &azObjectSorter{
		objects: objects,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(fs)
}

type azObjectSorter struct {
	objects []*container.BlobItem
	by      func(f1, f2 *container.BlobItem) bool // Closure used in the Less method.
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
