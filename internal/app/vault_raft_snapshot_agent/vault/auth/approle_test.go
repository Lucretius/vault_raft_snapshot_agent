package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultAppRoleAuth(t *testing.T) {
	authPath := "test"
	expectedRoleId := "testRoleId"
	expectedSecretId := "testSecretId"

	config := AppRoleAuthConfig{
		Path:     authPath,
		RoleId:   expectedRoleId,
		SecretId: expectedSecretId,
	}

	authApiStub := appRoleVaultAuthApiStub{}

	auth := createAppRoleAuth(config)
	_, err := auth.Refresh(&authApiStub)

	assert.NoError(t, err, "auth-refresh failed unexpectedly")
	assertAppRoleAuthValues(t, authPath, expectedRoleId, expectedSecretId, auth, authApiStub)
}

func assertAppRoleAuthValues(t *testing.T, expectedAuthPath string, expectedRoleId string, expectedSecretId string, auth authBackend, api appRoleVaultAuthApiStub) {
	t.Helper()

	assert.Equal(t, "AppRole", auth.name)
	assert.Equal(t, expectedAuthPath, api.path)
	assert.Equal(t, expectedRoleId, api.roleId)
	assert.Equal(t, expectedSecretId, api.secretId)
}

type appRoleVaultAuthApiStub struct {
	path     string
	roleId   string
	secretId string
}

func (stub *appRoleVaultAuthApiStub) LoginToBackend(path string, credentials map[string]interface{}) (leaseDuration time.Duration, err error) {
	stub.path = path
	stub.roleId = credentials["role_id"].(string)
	stub.secretId = credentials["secret_id"].(string)
	return 0, nil
}

func (stub *appRoleVaultAuthApiStub) LoginWithToken(token string) (leaseDuration time.Duration, err error) {
	return 0, errors.New("not implemented")
}
