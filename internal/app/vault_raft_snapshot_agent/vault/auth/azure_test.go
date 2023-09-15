package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/azure"

	"github.com/stretchr/testify/assert"
)

func TestCreateAzureAuth(t *testing.T) {
	config := AzureAuthConfig{
		Role: "test-role",
		Resource: "test-resource",
		Path: "test-path",
	}

	expectedAuthMethod, err := azure.NewAzureAuth(
		config.Role,
		azure.WithResource(config.Resource),
		azure.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewAzureAuth failed unexpectedly")

	auth, err := createAzureAuth(config)
	assert.NoError(t, err, "createAzureAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}
