package snapshot_agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

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
	API             *vaultApi.Client
	Uploader        *s3manager.Uploader
	S3Client        *s3.S3
	GCPBucket       *storage.BucketHandle
	AzureUploader   azblob.ContainerURL
	TokenExpiration time.Time
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
		err = snapshotter.ConfigureGCP(config)
		if err != nil {
			return nil, err
		}
	}
	if config.Azure.ContainerName != "" {
		err = snapshotter.ConfigureAzure(config)
		if err != nil {
			return nil, err
		}
	}
	return snapshotter, nil
}

func (s *Snapshotter) ConfigureVaultClient(config *config.Configuration) error {
	vaultConfig := vaultApi.DefaultConfig()
	if config.Address != "" {
		vaultConfig.Address = config.Address
	}
	tlsConfig := &vaultApi.TLSConfig{
		Insecure: true,
	}
	vaultConfig.ConfigureTLS(tlsConfig)
	api, err := vaultApi.NewClient(vaultConfig)
	if err != nil {
		return err
	}
	s.API = api
	err = s.SetClientToken(config)
	if err != nil {
		return err
	}
	return nil
}

func (s *Snapshotter) SetClientToken(config *config.Configuration) error {
	if config.TokenPath != "" {
		// The process generating this token file (e.g. a Vault Agent) should be
		// renewing or replacing the underlying token appropriately.
		cBytes, err := ioutil.ReadFile(config.TokenPath)
		if err != nil {
			return fmt.Errorf("error reading the given TokenPath: %s", err)
		}
		token := string(cBytes)
		s.API.SetToken(token)
		childInfo, err := s.API.Auth().Token().LookupSelf()
		if err != nil {
			s.API.ClearToken()
			return fmt.Errorf("error looking up provided token: %s", err)
		}
		ttl, err := childInfo.Data["ttl"].(json.Number).Int64()
		if err != nil {
			s.API.ClearToken()
			return fmt.Errorf("error converting ttl to int: %s", err)
		}
		s.TokenExpiration = time.Now().Add(time.Duration((time.Second * time.Duration(ttl)) / 2))
		return nil
	}
	data := map[string]interface{}{
		"role_id":   config.RoleID,
		"secret_id": config.SecretID,
	}
	approle := "approle"
	if config.Approle != "" {
		approle = config.Approle
	}
	resp, err := s.API.Logical().Write("auth/"+approle+"/login", data)
	if err != nil {
		return fmt.Errorf("error logging into AppRole auth backend: %s", err)
	}
	s.API.SetToken(resp.Auth.ClientToken)
	s.TokenExpiration = time.Now().Add(time.Duration((time.Second * time.Duration(resp.Auth.LeaseDuration)) / 2))
	return nil
}

func (s *Snapshotter) ConfigureS3(config *config.Configuration) error {
	awsConfig := &aws.Config{Region: aws.String(config.AWS.Region)}

	if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(config.AWS.AccessKeyID, config.AWS.SecretAccessKey, "")
	}

	if config.AWS.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.AWS.Endpoint)
	}

	if config.AWS.S3ForcePathStyle != false {
		awsConfig.S3ForcePathStyle = aws.Bool(config.AWS.S3ForcePathStyle)
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
