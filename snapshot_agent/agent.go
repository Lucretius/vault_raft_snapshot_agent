package snapshot_agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Lucretius/vault_raft_snapshot_agent/config"
	vaultApi "github.com/hashicorp/vault/api"
)

type UploaderType string

const (
	S3UploaderType    UploaderType = "s3"
	GCPUploaderType                = "gcp"
	AzureUploaderType              = "azure"
	LocalUploaderType              = "local"
)

type Uploader interface {
	Upload(ctx context.Context, reader io.Reader, currentTs int64) (string, error)
	LastSuccessfulUpload(ctx context.Context) (time.Time, error)
}

type Snapshotter struct {
	API *vaultApi.Client
	sync.Mutex
	Uploaders       map[UploaderType]Uploader
	TokenExpiration time.Time
}

func NewSnapshotter(config *config.Configuration) (*Snapshotter, error) {
	snapshotter := &Snapshotter{
		Uploaders: map[UploaderType]Uploader{},
	}
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
	if config.Local.Path != "" {
		err = snapshotter.ConfigureLocal(config)
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
	err := vaultConfig.ConfigureTLS(tlsConfig)
	if err != nil {
		return err
	}
	api, err := vaultApi.NewClient(vaultConfig)
	if err != nil {
		return err
	}
	s.API = api

	switch config.VaultAuthMethod {
	case "k8s":
		return s.SetClientTokenFromK8sAuth(config)
	case "token":
		return s.SetClientToken(config)
	default:
		return s.SetClientTokenFromAppRole(config)
	}
}

func (s *Snapshotter) SetClientToken(config *config.Configuration) error {
	s.API.SetToken(config.Token)

	tokenInfo, err := s.API.Auth().Token().LookupSelf()

	if err != nil {
		s.API.ClearToken()
		return fmt.Errorf("error looking up provided token: %s", err)
	}

	ttl, err := tokenInfo.Data["ttl"].(json.Number).Int64()
	if err != nil {
		s.API.ClearToken()
		return fmt.Errorf("error converting ttl to int: %s", err)
	}

	s.TokenExpiration = time.Now().Add((time.Second * time.Duration(ttl)) / 2)
	return nil
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
	s.TokenExpiration = time.Now().Add((time.Second * time.Duration(resp.Auth.LeaseDuration)) / 2)
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
	data := map[string]interface{}{
		"role": config.K8sAuthRole,
		"jwt":  string(jwt),
	}

	login := path.Clean("auth/" + config.K8sAuthPath + "/login")
	result, err := s.API.Logical().Write(login, data)
	if err != nil {
		return err
	}

	s.API.SetToken(result.Auth.ClientToken)
	s.TokenExpiration = time.Now().Add((time.Second * time.Duration(result.Auth.LeaseDuration)) / 2)
	return nil
}

func (s *Snapshotter) ConfigureS3(config *config.Configuration) error {
	uploader, err := NewS3Uploader(config)

	if err != nil {
		return fmt.Errorf("unable to create S3 uploader: %w", err)
	}

	s.Lock()
	defer s.Unlock()

	s.Uploaders[S3UploaderType] = uploader

	return nil
}

func (s *Snapshotter) ConfigureGCP(config *config.Configuration) error {
	uploader, err := NewGCPUploader(config)

	if err != nil {
		return fmt.Errorf("unable to create GCP uploader: %w", err)
	}

	s.Lock()
	defer s.Unlock()

	s.Uploaders[GCPUploaderType] = uploader

	return nil
}

func (s *Snapshotter) ConfigureAzure(config *config.Configuration) error {
	uploader, err := NewAzureUploader(config)

	if err != nil {
		return fmt.Errorf("unable to create Azure uploader: %w", err)
	}

	s.Lock()
	defer s.Unlock()

	s.Uploaders[AzureUploaderType] = uploader

	return nil
}

func (s *Snapshotter) ConfigureLocal(config *config.Configuration) error {
	uploader, err := NewLocalUploader(config)

	if err != nil {
		return fmt.Errorf("unable to create local uploader: %w", err)
	}

	s.Lock()
	defer s.Unlock()

	s.Uploaders[LocalUploaderType] = uploader

	return nil
}

func (s *Snapshotter) GetLastSuccessfulUploads(ctx context.Context) (LastUpload, error) {
	lastSuccessfulUpload := LastUpload{}

	s.Lock()
	defer s.Unlock()

	for uploaderType, uploader := range s.Uploaders {
		lastUpload, err := uploader.LastSuccessfulUpload(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to get last successful upload for %s: %w", uploaderType, err)
		}
		lastSuccessfulUpload[uploaderType] = lastUpload
	}

	return lastSuccessfulUpload, nil
}
