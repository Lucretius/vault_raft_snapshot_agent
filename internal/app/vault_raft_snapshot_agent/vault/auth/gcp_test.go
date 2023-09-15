package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/gcp"

	"github.com/stretchr/testify/assert"
)

func TestCreateGCPGCEAuth(t *testing.T) {
	config := GCPAuthConfig{
		Role: "test-role",
		Path: "test-path",
	}

	expectedAuthMethod, err := gcp.NewGCPAuth(
		config.Role,
		gcp.WithGCEAuth(),
		gcp.WithMountPath("test-path"),
	)
	assert.NoError(t, err, "NewGCPAuth failed unexpectedly")

	auth, err := createGCPAuth(config)
	assert.NoError(t, err, "createGCPAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}

func TestCreateGCPIAMAuth(t *testing.T) {
	config := GCPAuthConfig{
		Role: "test-role",
		ServiceAccountEmail: "test@email.com",
		Path: "test-path",
	}

	expectedAuthMethod, err := gcp.NewGCPAuth(
		config.Role,
		gcp.WithIAMAuth(config.ServiceAccountEmail),
		gcp.WithMountPath("test-path"),
	)
	assert.NoError(t, err, "NewGCPAuth failed unexpectedly")

	auth, err := createGCPAuth(config)
	assert.NoError(t, err, "createGCPAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
