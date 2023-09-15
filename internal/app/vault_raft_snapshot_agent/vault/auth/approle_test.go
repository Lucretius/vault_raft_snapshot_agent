package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/approle"

	"github.com/stretchr/testify/assert"
)

func TestCreateAppRoleAuth(t *testing.T) {
	config := AppRoleAuthConfig{
		RoleId:   "test-role",
		SecretId: "test-secret",
		Path:     "test-path",
	}

	expectedAuthMethod, err := approle.NewAppRoleAuth(
		config.RoleId,
		&approle.SecretID{FromString: config.SecretId},
		approle.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewAppRoleAuth failed unexpectedly")

	auth, err := createAppRoleAuth(config)
	assert.NoError(t, err, "createAppRoleAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
