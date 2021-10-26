package snapshot_agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
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

	if config.Timeout != "" {
		t, err := time.ParseDuration(config.Timeout)
		if err == nil {
			vaultConfig.Timeout = t
		}
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
	if config.VaultAuthMethod == "k8s" {
		return s.SetClientTokenFromK8sAuth(config)
	}
	return s.SetClientTokenFromAppRole(config)
}

func (s *Snapshotter) SetClientTokenFromAppRole(config *config.Configuration) error {
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

func (s *Snapshotter) SetClientTokenFromK8sAuth(config *config.Configuration) error {

	if config.K8sAuthPath == "" || config.K8sAuthRole == "" {
		return errors.New("missing k8s auth definitions")
	}

	jwt, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return err
	}
	data := map[string]string{
		"role": config.K8sAuthRole,
		"jwt":  string(jwt),
	}

	login := path.Clean("/v1/auth/" + config.K8sAuthPath + "/login")
	req := s.API.NewRequest("POST", login)
	req.SetJSONBody(data)

	resp, err := s.API.RawRequest(req)
	if err != nil {
		return err
	}
	if respErr := resp.Error(); respErr != nil {
		return respErr
	}

	var result vaultApi.Secret
	if err := resp.DecodeJSON(&result); err != nil {
		return err
	}

	s.API.SetToken(result.Auth.ClientToken)
	s.TokenExpiration = time.Now().Add(time.Duration((time.Second * time.Duration(result.Auth.LeaseDuration)) / 2))
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
