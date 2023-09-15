package auth

import (
	"fmt"
	"testing"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/config"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/test"

	"github.com/hashicorp/vault/api/auth/kubernetes"

	"github.com/stretchr/testify/assert"
)

func TestCreateKubernetesAuth(t *testing.T) {
	jwtPath := fmt.Sprintf("%s/jwt", t.TempDir())
	config := KubernetesAuthConfig{
		Role: "test-role",
		JWTPath: config.Path(jwtPath),
		Path: "test-path",
	}

	err := test.WriteFile(t, jwtPath, "test")
	assert.NoError(t, err, "could not write jwt-file")

	expectedAuthMethod, err := kubernetes.NewKubernetesAuth(
		config.Role,
		kubernetes.WithMountPath(config.Path),
		kubernetes.WithServiceAccountTokenPath(string(config.JWTPath)),
	)
	assert.NoError(t, err, "NewKubernetesAuth failed unexpectedly")

	auth, err := createKubernetesAuth(config)
	assert.NoError(t, err, "createKubernetesAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
