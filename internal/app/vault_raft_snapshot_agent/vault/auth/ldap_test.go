package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/ldap"

	"github.com/stretchr/testify/assert"
)

func TestCreateLDAPAuth(t *testing.T) {
	config := LDAPAuthConfig{
		Username: "test-user",
		Password: "test-password",
		Path: "test-path",
	}

	expectedAuthMethod, err := ldap.NewLDAPAuth(
		config.Username,
		&ldap.Password{FromString: config.Password},
		ldap.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewLDAPAuth failed unexpectedly")

	auth, err := createLDAPAuth(config)
	assert.NoError(t, err, "createLDAPAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
