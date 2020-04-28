package snapshot_agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	vaultApi "github.com/hashicorp/vault/api"
)

type Snapshotter struct {
	API           *vaultApi.Client
	Uploader      *s3manager.Uploader
	S3Client      *s3.S3
	GCPBucket     *storage.BucketHandle
	AzureUploader azblob.ContainerURL
}

func NewSnapshotter(config *config.Configuration) (*Snapshotter, error) {
	snapshotter := &Snapshotter{}
	err := snapshotter.ConfigureVaultClient(config)
	if err != nil {
		return nil, err
	}
	if config.AWS.Bucket != "" {
		err = snapshotter.ConfigureS3(config)
		if err != nil {
			return nil, err
		}
	}
	if config.GCP.Bucket != "" {
		err = snapshotter.ConfigureS3(config)
		if err != nil {
			return nil, err
		}
	}
	return snapshotter, nil
}

func (s *Snapshotter) ConfigureVaultClient(config *config.Configuration) error {
	vaultConfig := vaultApi.DefaultConfig()
	tokenEnvVar := os.Getenv("SNAPSHOT_TOKEN")
	if tokenEnvVar != "" {
		config.Token = tokenEnvVar
	}
	vaultConfig.Address = config.Address
	tlsConfig := &vaultApi.TLSConfig{
		Insecure: true,
	}
	vaultConfig.ConfigureTLS(tlsConfig)
	api, err := vaultApi.NewClient(vaultConfig)
	if err != nil {
		return err
	}
	api.SetToken(config.Token)
	s.API = api
	return nil
}

func (s *Snapshotter) ConfigureS3(config *config.Configuration) error {
	awsConfig := &aws.Config{Region: aws.String(config.AWS.Region)}

	if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(config.AWS.AccessKeyID, config.AWS.SecretAccessKey, "")
	}

	sess := session.Must(session.NewSession(awsConfig))
	s.S3Client = s3.New(sess)
	s.Uploader = s3manager.NewUploader(sess)
	return nil
}

func (s *Snapshotter) ConfigureGCP(config *config.Configuration) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	s.GCPBucket = client.Bucket(config.GCP.Bucket)
	return nil
}

func (s *Snapshotter) ConfigureAzure(config *config.Configuration) error {
	accountName := config.Azure.AccountName
	if os.Getenv("AZURE_STORAGE_ACCOUNT") != "" {
		accountName = os.Getenv("AZURE_STORAGE_ACCOUNT")
	}
	accountKey := config.Azure.AccountKey
	if os.Getenv("AZURE_STORAGE_ACCESS_KEY") != "" {
		accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	}
	if len(accountName) == 0 || len(accountKey) == 0 {
		return errors.New("Invalid Azure configuration")
	}
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, config.Azure.ContainerName))

	s.AzureUploader = azblob.NewContainerURL(*URL, p)
	return nil
}
