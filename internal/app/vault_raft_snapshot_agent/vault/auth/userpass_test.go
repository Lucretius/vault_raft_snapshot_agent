package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/userpass"

	"github.com/stretchr/testify/assert"
)

func TestCreateUserpassAuth(t *testing.T) {
	config := UserPassAuthConfig{
		Username: "test-user",
		Password: "test-password",
		Path: "test-path",
	}

	expectedAuthMethod, err := userpass.NewUserpassAuth(
		config.Username,
		&userpass.Password{FromString: config.Password},
		userpass.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewUserPassAuth failed unexpectedly")

	auth, err := createUserPassAuth(config)
	assert.NoError(t, err, "createUserpassAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
