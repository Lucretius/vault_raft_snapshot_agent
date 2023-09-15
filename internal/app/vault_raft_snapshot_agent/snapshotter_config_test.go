package vault_raft_snapshot_agent

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/config"
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

func relativeTo(configFile string, file string) config.Path {
	if !filepath.IsAbs(file) && !strings.HasPrefix(file, "/") {
		file = filepath.Join(filepath.Dir(configFile), file)
	}

	if !filepath.IsAbs(file) && !strings.HasPrefix(file, "/") {
		file, _ = filepath.Abs(file)
		file = filepath.Clean(file)
	}

	return config.Path(file)
}

func TestReadCompleteConfig(t *testing.T) {
	configFile := "../../../testdata/complete.yaml"

	expectedConfig := SnapshotterConfig{
		Vault: vault.VaultClientConfig{
			Url:      "https://example.com:8200",
			Insecure: true,
			Timeout:  5 * time.Minute,
			Auth: auth.AuthConfig{
				AppRole: auth.AppRoleAuthConfig{
					Path:     "test-approle-path",
					RoleId:   "test-approle",
					SecretId: "test-approle-secret",
				},
				AWS: auth.AWSAuthConfig{
					Path:             "test-aws-path",
					Role:             "test-aws-role",
					Region:           "test-region",
					EC2Nonce:         "test-nonce",
					EC2SignatureType: auth.AWS_EC2_RSA2048,
				},
				Azure: auth.AzureAuthConfig{
					Path:     "test-azure-path",
					Role:     "test-azure-role",
					Resource: "test-resource",
				},
				GCP: auth.GCPAuthConfig{
					Path:                "test-gcp-path",
					Role:                "test-gcp-role",
					ServiceAccountEmail: "test@example.com",
				},
				Kubernetes: auth.KubernetesAuthConfig{
					Role:    "test-kubernetes-role",
					Path:    "test-kubernetes-path",
					JWTPath: relativeTo(configFile, defaultJwtPath("./jwt")),
				},
				LDAP: auth.LDAPAuthConfig{
					Path:     "test-ldap-path",
					Username: "test-ldap-user",
					Password: "test-ldap-pass",
				},
				Token: "test-token",
				UserPass: auth.UserPassAuthConfig{
					Path:     "test-userpass-path",
					Username: "test-user",
					Password: "test-pass",
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
			AWS: upload.AWSUploaderConfig{
				Endpoint:                "test-endpoint",
				Region:                  "test-region",
				Bucket:                  "test-bucket",
				KeyPrefix:               "test-prefix",
				UseServerSideEncryption: true,
				ForcePathStyle:          true,
				Credentials: upload.AWSUploaderCredentialsConfig{
					Key:    "test-key",
					Secret: "test-secret",
				},
			},
			Azure: upload.AzureUploaderConfig{
				AccountName:   "test-account",
				AccountKey:    "test-key",
				ContainerName: "test-container",
				CloudDomain:   "blob.core.chinacloudapi.cn",
			},
			GCP: upload.GCPUploaderConfig{
				Bucket: "test-bucket",
			},
			Local: upload.LocalUploaderConfig{
				Path: ".",
			},
			Swift: upload.SwiftUploaderConfig{
				Container: "test-container",
				UserName:  "test-username",
				ApiKey:    "test-api-key",
				AuthUrl:   "http://auth.com",
				Domain:    "http://user.com",
				Region:    "test-region",
				TenantId:  "test-tenant",
				Timeout:   180 * time.Second,
			},
		},
	}

	data := SnapshotterConfig{}
	parser := config.NewParser[*SnapshotterConfig]("VRSA", "")
	err := parser.ReadConfig(&data, configFile)

	assert.NoError(t, err, "ReadConfig(%s) failed unexpectedly", configFile)
	assert.Equal(t, expectedConfig, data)
}

func TestReadConfigSetsDefaultValues(t *testing.T) {
	configFile := "../../../testdata/defaults.yaml"

	expectedConfig := SnapshotterConfig{
		Vault: vault.VaultClientConfig{
			Url:      "http://127.0.0.1:8200",
			Insecure: false,
			Timeout:  time.Minute,
			Auth: auth.AuthConfig{
				AppRole: auth.AppRoleAuthConfig{
					Path:  "approle",
					Empty: true,
				},
				AWS: auth.AWSAuthConfig{
					Path:             "aws",
					EC2SignatureType: auth.AWS_EC2_PKCS7,
					Empty:            true,
				},
				Azure: auth.AzureAuthConfig{
					Path:  "azure",
					Empty: true,
				},
				GCP: auth.GCPAuthConfig{
					Path:  "gcp",
					Empty: true,
				},
				Kubernetes: auth.KubernetesAuthConfig{
					Role:    "test-role",
					Path:    "kubernetes",
					JWTPath: relativeTo(configFile, defaultJwtPath("")),
				},
				LDAP: auth.LDAPAuthConfig{
					Path:  "ldap",
					Empty: true,
				},
				UserPass: auth.UserPassAuthConfig{
					Path:  "userpass",
					Empty: true,
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
			AWS: upload.AWSUploaderConfig{
				Credentials: upload.AWSUploaderCredentialsConfig{Empty: true},
				Empty:       true,
			},
			Azure: upload.AzureUploaderConfig{
				CloudDomain: "blob.core.windows.net",
				Empty:       true,
			},
			GCP: upload.GCPUploaderConfig{Empty: true},
			Local: upload.LocalUploaderConfig{
				Path: ".",
			},
			Swift: upload.SwiftUploaderConfig{
				Timeout: time.Minute,
				Empty:       true,
			},
		},
	}

	data := SnapshotterConfig{}
	parser := config.NewParser[*SnapshotterConfig]("VRSA", "")
	err := parser.ReadConfig(&data, configFile)

	assert.NoError(t, err, "ReadConfig(%s) failed unexpectedly", configFile)
	assert.Equal(t, expectedConfig, data)
}

func init() {
	jwtPath := defaultJwtPath("")
	if err := os.MkdirAll(filepath.Dir(jwtPath), 0777); err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatalf("could not create directorys for jwt-file %s: %v", jwtPath, err)
	}

	file, err := os.OpenFile(jwtPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("could not create jwt-file %s: %v", jwtPath, err)
	}

	file.Close()

	if err != nil {
		log.Fatalf("could not read jwt-file %s: %v", jwtPath, err)
	}
}
