package vault_raft_snapshot_agent

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/upload"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault/auth"
	"github.com/stretchr/testify/assert"
)

// allow overiding "default" kubernetes-jwt-path so that tests on ci do not fail
func defaultJwtPath(def string) string {
	jwtPath := os.Getenv("VRSA_VAULT_AUTH_KUBERNETES_JWTPATH")
	if jwtPath != "" {
		return jwtPath
	}

	if def != "" {
		return def
	}

	return "/var/run/secrets/kubernetes.io/serviceaccount/token"
}

func TestReadEmptyConfig(t *testing.T) {
	file := "../../../testdata/empty.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for empty file`, file)
}

func TestReadConfigWithInvalidAddr(t *testing.T) {
	file := "../../../testdata/invalid-url.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for config with invalid url`, file)
}

func TestReadConfigWithoutUploaders(t *testing.T) {
	file := "../../../testdata/no-uploaders.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for config without uploaders`, file)
}

func TestReadConfigWithInvalidUploader(t *testing.T) {
	file := "../../../testdata/invalid-uploader.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for config with invalid uploader`, file)
}

func TestReadConfigWithInvalidLocalUploadPath(t *testing.T) {
	file := "../../../testdata/invalid-local-upload-path.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for config with invalid local upload-path`, file)
}

func TestReadConfigWithInvalidAuth(t *testing.T) {
	file := "../../../testdata/invalid-auth.yaml"
	_, err := ReadConfig(file)

	assert.Error(t, err, `ReadConfig(%s) should return error for config with invalid auth`, file)
}

func TestReadCompleteConfig(t *testing.T) {
	expectedConfig := SnapshotterConfig{
		Vault: vault.VaultClientConfig{
			Url:      "https://example.com:8200",
			Insecure: true,
			Auth: auth.AuthConfig{
				AppRole: auth.AppRoleAuthConfig{
					Path:  "approle",
					Empty: true,
				},
				Kubernetes: auth.KubernetesAuthConfig{
					Role:    "test-role",
					Path:    "test-auth",
					JWTPath: defaultJwtPath("./jwt"),
				},
			},
		},
		Snapshots: SnapshotConfig{
			Frequency:       time.Hour * 2,
			Retain:          10,
			Timeout:         time.Minute * 2,
			NamePrefix:      "test-",
			NameSuffix:      ".test",
			TimestampFormat: "2006-01-02",
		},
		Uploaders: upload.UploadersConfig{
			AWS: upload.AWSConfig{
				Endpoint:                "test-endpoint",
				Region:                  "test-region",
				Bucket:                  "test-bucket",
				KeyPrefix:               "test-prefix",
				UseServerSideEncryption: true,
				ForcePathStyle:          true,
				Credentials: upload.AWSCredentialsConfig{
					Key:    "test-key",
					Secret: "test-secret",
				},
			},
			Azure: upload.AzureConfig{
				AccountName:   "test-account",
				AccountKey:    "test-key",
				ContainerName: "test-container",
			},
			GCP: upload.GCPConfig{
				Bucket: "test-bucket",
			},
			Local: upload.LocalConfig{
				Path: ".",
			},
		},
	}

	file := "../../../testdata/complete.yaml"
	config, err := ReadConfig(file)

	assert.NoError(t, err, "ReadConfig(%s) failed unexpectedly", file)
	assert.Equal(t, expectedConfig, config)
}

func TestReadConfigSetsDefaultValues(t *testing.T) {
	expectedConfig := SnapshotterConfig{
		Vault: vault.VaultClientConfig{
			Url:      "http://127.0.0.1:8200",
			Insecure: false,
			Auth: auth.AuthConfig{
				AppRole: auth.AppRoleAuthConfig{
					Path:  "approle",
					Empty: true,
				},
				Kubernetes: auth.KubernetesAuthConfig{
					Role:    "test-role",
					Path:    "kubernetes",
					JWTPath: defaultJwtPath(""),
				},
			},
		},
		Snapshots: SnapshotConfig{
			Frequency:       time.Hour,
			Retain:          0,
			Timeout:         time.Minute,
			NamePrefix:      "raft-snapshot-",
			NameSuffix:      ".snap",
			TimestampFormat: "2006-01-02T15-04-05Z-0700",
		},
		Uploaders: upload.UploadersConfig{
			AWS: upload.AWSConfig{
				Credentials: upload.AWSCredentialsConfig{Empty: true},
				Empty:       true,
			},
			Azure: upload.AzureConfig{Empty: true},
			GCP:   upload.GCPConfig{Empty: true},
			Local: upload.LocalConfig{
				Path: ".",
			},
		},
	}

	file := "../../../testdata/defaults.yaml"
	config, err := ReadConfig(file)

	assert.NoError(t, err, "ReadConfig(%s) failed unexpectedly", file)
	assert.Equal(t, expectedConfig, config)
}

func TestReadConfigBindsEnvVariables(t *testing.T) {
	t.Setenv("VAULT_ADDR", "http://from.env:8200")
	t.Setenv("AWS_ACCESS_KEY_ID", "env-key")
	t.Setenv("SECRET_ACCESS_KEY", "env-secret")
	t.Setenv("VRSA_VAULT_AUTH_KUBERNETES_ROLE", "test")
	t.Setenv("VRSA_VAULT_AUTH_KUBERNETES_JWTPATH", "./jwt")

	file := "../../../testdata/envvars.yaml"
	config, err := ReadConfig(file)
	assert.NoError(t, err, "ReadConfig(%s) failed unexpectedly", file)

	assert.Equal(t, os.Getenv("VAULT_ADDR"), config.Vault.Url, "ReadConfig did not bind env-var VAULT_ADDR")
	assert.Equal(t, os.Getenv("AWS_ACCESS_KEY_ID"), config.Uploaders.AWS.Credentials.Key, "ReadConfig did not bind env-var AWS_ACCESS_KEY_ID")
	assert.Equal(t, os.Getenv("SECRET_ACCESS_KEY"), config.Uploaders.AWS.Credentials.Secret, "ReadConfig did not bind env-var SECRET_ACCESS_KEY")
	assert.Equal(t, os.Getenv("VRSA_VAULT_AUTH_KUBERNETES_JWTPATH"), config.Vault.Auth.Kubernetes.JWTPath, "ReadConfig did not bind env-var VRSA_VAULT_AUTH_KUBERNETES_JWTPATH")

}

func TestWatchAndReConfigure(t *testing.T) {
	tempDir := t.TempDir()
	file1 := "../../../testdata/watch-and-reconfigure1.yaml"
	file2 := "../../../testdata/watch-and-reconfigure2.yaml"
	configFile := fmt.Sprintf("%s/config.yaml", tempDir)

	err := copyFile(t, "../../../testdata/jwt", fmt.Sprintf("%s/jwt", tempDir))
	assert.NoError(t, err, "could not copy file jwt-file")

	err = copyFile(t, file1, configFile)
	assert.NoError(t, err, "could not copy file %s", file1)

	config, err := ReadConfig(configFile)
	assert.NoError(t, err, "could not read config-file %s", file1)

	snapshotter, err := CreateSnapshotter(config)
	assert.NoError(t, err, "could not create snapshotter")
	assert.Equal(t, 30 * time.Second, snapshotter.config.Frequency)

	reconfigured := WatchConfigAndReconfigure(snapshotter)

	errs := make(chan error, 1)
	go func() {
		errs <- copyFile(t, file2, configFile)
	}()

	assert.NoError(t, <-errs, "could not copy file %s", file2)
	assert.NoError(t, <-reconfigured)
	assert.Equal(t, time.Minute, snapshotter.config.Frequency)

	parser.OnConfigChange(func() { /* prevent error messages on cleanup */ })
}

func copyFile(t *testing.T, source string, dest string) error {
	t.Helper()

	in, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp", dest)
	if err := os.WriteFile(tmpFile, in, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, dest)
}

func init() {
	jwtPath := defaultJwtPath("")
	if err := os.MkdirAll(filepath.Dir(jwtPath), 0777); err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatalf("could not create directorys for jwt-file %s: %v", jwtPath, err)
	}

	file, err := os.OpenFile(defaultJwtPath(""), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("could not create jwt-file %s: %v", jwtPath, err)
	}

	file.Close()

	if err != nil {
		log.Fatalf("could not read jwt-file %s: %v", jwtPath, err)
	}
}
