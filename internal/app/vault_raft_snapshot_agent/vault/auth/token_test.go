package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateTokenAuth(t *testing.T) {
	expectedToken := "test"
	authApiStub := tokenVaultAuthApiStub{}

	auth := createTokenAuth(expectedToken)

	_, err := auth.Refresh(&authApiStub)

	assert.NoError(t, err, "token-auth failed unexpectedly")
	assert.Equal(t, expectedToken, authApiStub.token)
}

func TestTokenAuthFailsIfLoginFails(t *testing.T) {
	authApiStub := tokenVaultAuthApiStub{loginFails: true}
	auth := createTokenAuth("test")

	_, err := auth.Refresh(&authApiStub)

	assert.Errorf(t, err, "token-auth did not report error although login failed!")
}

func TestTokenAuthReturnsExpirationBasedOnLoginLeaseDuration(t *testing.T) {
	authApiStub := tokenVaultAuthApiStub{leaseDuration: time.Minute}

	auth := createTokenAuth("test")

	expiration, err := auth.Refresh(&authApiStub)

	assert.NoErrorf(t, err, "token-auth failed unexpectedly")

	expectedExpiration := time.Now().Add((time.Second * authApiStub.leaseDuration) / 2)
	assert.WithinDuration(t, expectedExpiration, expiration, time.Millisecond)
}

type tokenVaultAuthApiStub struct {
	token         string
	loginFails    bool
	leaseDuration time.Duration
}

func (stub *tokenVaultAuthApiStub) LoginToBackend(path string, credentials map[string]interface{}) (leaseDuration time.Duration, err error) {
	return 0, errors.New("not implemented")
}

func (stub *tokenVaultAuthApiStub) LoginWithToken(token string) (leaseDuration time.Duration, err error) {
	stub.token = token
	if stub.loginFails {
		return 0, errors.New("login failed")
	}
	return stub.leaseDuration, nil
}
