package upload

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type AzureConfig struct {
	AccountName   string `validate:"required_if=Empty false"`
	AccountKey    string `validate:"required_if=Empty false"`
	ContainerName string `mapstructure:"container" validate:"required_if=Empty false"`
	Empty         bool
}

type azureUploaderImpl struct {
	client    *azblob.Client
	container string
}

func createAzureUploader(config AzureConfig) (*uploader[*container.BlobItem], error) {
	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials for azure: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.AccountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure client: %w", err)
	}

	return &uploader[*container.BlobItem]{
		azureUploaderImpl{
			client:    client,
			container: config.ContainerName,
		},
	}, nil
}

func (u azureUploaderImpl) Destination() string {
	return fmt.Sprintf("azure container %s", u.container)
}

// nolint:unused
// implements interface uploaderImpl
func (u azureUploaderImpl) uploadSnapshot(ctx context.Context, name string, data io.Reader) error {
	uploadOptions := &azblob.UploadStreamOptions{
		BlockSize:   4 * 1024 * 1024,
		Concurrency: 16,
	}

	if _, err := u.client.UploadStream(ctx, u.container, name, data, uploadOptions); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u azureUploaderImpl) deleteSnapshot(ctx context.Context, snapshot *container.BlobItem) error {
	if _, err := u.client.DeleteBlob(ctx, u.container, *snapshot.Name, nil); err != nil {
		return err
	}

	return nil
}

// nolint:unused
// implements interface uploaderImpl
func (u azureUploaderImpl) listSnapshots(ctx context.Context, prefix string, ext string) ([]*container.BlobItem, error) {
	var results []*container.BlobItem

	var maxResults int32 = 500

	pager := u.client.NewListBlobsFlatPager(u.container, &azblob.ListBlobsFlatOptions{
		Prefix:     &prefix,
		MaxResults: &maxResults,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)

		if err != nil {
			return results, err
		}

		results = append(results, resp.Segment.BlobItems...)
	}

	return results, nil
}

// nolint:unused
// implements interface uploaderImpl
func (u azureUploaderImpl) compareSnapshots(a, b *container.BlobItem) int {
	return a.Properties.LastModified.Compare(*b.Properties.LastModified)
}
