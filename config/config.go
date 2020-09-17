package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Configuration is the overall config object
type Configuration struct {
	Address  string      `json:"addr"`
	Retain   int64       `json:"retain"`
	Frequency string      `json:"frequency"`
	AWS      S3Config    `json:"aws_storage"`
	Local    LocalConfig `json:"local_storage"`
	GCP      GCPConfig   `json:"google_storage"`
	Azure    AzureConfig `json:"azure_storage"`
	RoleID   string      `json:"role_id"`
	SecretID string      `json:"secret_id"`
}

// AzureConfig is the configuration for Azure blob snapshots
type AzureConfig struct {
	AccountName   string `json:"account_name"`
	AccountKey    string `json:"account_key"`
	ContainerName string `json:"container_name"`
}

// GCPConfig is the configuration for GCP Storage snapshots
type GCPConfig struct {
	Bucket string `json:"bucket"`
}

// LocalConfig is the configuration for local snapshots
type LocalConfig struct {
	Path string `json:"path"`
}

// S3Config is the configuration for S3 snapshots
type S3Config struct {
	Uploader           *s3manager.Uploader
	AccessKeyID        string `json:"access_key_id"`
	SecretAccessKey    string `json:"secret_access_key"`
	Endpoint           string `json:"s3_endpoint"`
	Region             string `json:"s3_region"`
	Bucket             string `json:"s3_bucket"`
	KeyPrefix          string `json:"s3_key_prefix"`
	SSE                bool   `json:"s3_server_side_encryption"`
	StaticSnapshotName string `json:"s3_static_snapshot_name"`
	S3ForcePathStyle   bool   `json:"s3_force_path_style"`
}

// ReadConfig reads the configuration file
func ReadConfig() (*Configuration, error) {
	file := "/etc/vault.d/snapshot.json"
	if len(os.Args) > 1 {
		file = os.Args[1]
	}
	cBytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("Cannot read configuration file: %v", err.Error())
	}
	c := &Configuration{}
	err = json.Unmarshal(cBytes, &c)
	if err != nil {
		log.Fatalf("Cannot parse configuration file: %v", err.Error())
	}
	return c, nil
}
