package auth

import (
	"errors"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
)

func TestCreateTokenAuth(t *testing.T) {
	expectedToken := "test"
	authApiStub := tokenVaultAuthApiStub{}

	auth := createTokenAuth(expectedToken)

	_, err := auth.login(&authApiStub)

	assert.NoError(t, err, "token-auth failed unexpectedly")
	assert.Equal(t, expectedToken, authApiStub.token)
}

func TestTokenAuthFailsIfLoginFails(t *testing.T) {
	authApiStub := tokenVaultAuthApiStub{loginFails: true}
	auth := createTokenAuth("test")

	_, err := auth.login(&authApiStub)

	assert.Error(t, err, "token-auth did not report error although login failed!")
}

func TestTokenAuthReturnsExpirationBasedOnLoginLeaseDuration(t *testing.T) {
	authApiStub := tokenVaultAuthApiStub{leaseDuration: 60}

	auth := createTokenAuth("test")

	leaseDuration, err := auth.login(&authApiStub)

	assert.NoErrorf(t, err, "token-auth failed unexpectedly")

	expectedDuration := time.Duration(authApiStub.leaseDuration)
	assert.Equal(t, expectedDuration, leaseDuration, time.Millisecond)
}

type tokenVaultAuthApiStub struct {
	token         string
	loginFails    bool
	leaseDuration int64
}

func (stub *tokenVaultAuthApiStub) SetToken(token string) {
	stub.token = token
}

func (stub *tokenVaultAuthApiStub) ClearToken() {
	stub.token = ""
}

func (stub *tokenVaultAuthApiStub) LookupToken() (*api.Secret, error) {
	if stub.loginFails {
		return &api.Secret{}, errors.New("lookup failed")
	}

	return &api.Secret{
		Data: map[string]interface{}{
			"ttl": json.Number(fmt.Sprint(stub.leaseDuration)),
		},
	}, nil
}
