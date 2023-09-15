package auth

import (
	"testing"

	"github.com/hashicorp/vault/api/auth/aws"

	"github.com/stretchr/testify/assert"
)

func TestCreateAWSIAMAuth(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-key")
	t.Setenv("AWS_SESSION_TOKEN", "test-token")

	config := AWSAuthConfig{
		Role:              "test-role",
		IAMServerIDHeader: "test-header",
		Region:            "test-region",
		Path:              "test-path",
	}

	expectedAuthMethod, err := aws.NewAWSAuth(
		aws.WithRole(config.Role),
		aws.WithIAMAuth(),
		aws.WithIAMServerIDHeader(config.IAMServerIDHeader),
		aws.WithRegion(config.Region),
		aws.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewAWSAuth failed unexpectedly")

	auth, err := createAWSAuth(config)
	assert.NoError(t, err, "createAWSAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}

func TestCreateAWSEC2DefaultAuth(t *testing.T) {
	config := AWSAuthConfig{
		Role:     "test-role",
		EC2Nonce: "test-nonce",
		Region:   "test-region",
		Path:     "test-path",
	}

	expectedAuthMethod, err := aws.NewAWSAuth(
		aws.WithRole(config.Role),
		aws.WithEC2Auth(),
		aws.WithNonce(config.EC2Nonce),
		aws.WithPKCS7Signature(),
		aws.WithRegion(config.Region),
		aws.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewAWSAuth failed unexpectedly")

	auth, err := createAWSAuth(config)
	assert.NoError(t, err, "createAWSAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}

func TestCreateAWSEC2RSA2048Auth(t *testing.T) {
	config := AWSAuthConfig{
		Role:             "test-role",
		EC2Nonce:         "test-nonce",
		EC2SignatureType: "rsa2048",
		Region:           "test-region",
		Path:             "test-path",
	}

	expectedAuthMethod, err := aws.NewAWSAuth(
		aws.WithRole(config.Role),
		aws.WithEC2Auth(),
		aws.WithNonce(config.EC2Nonce),
		aws.WithRSA2048Signature(),
		aws.WithRegion(config.Region),
		aws.WithMountPath(config.Path),
	)
	assert.NoError(t, err, "NewAWSAuth failed unexpectedly")

	auth, err := createAWSAuth(config)
	assert.NoError(t, err, "createAWSAuth failed unexpectedly")

	assert.Equal(t, expectedAuthMethod, auth.delegate)
}

func TestCreateAWSEC2AuthFailsForUnknownSignatureType(t *testing.T) {
	config := AWSAuthConfig{
		Role:             "test-role",
		EC2Nonce:         "test-nonce",
		EC2SignatureType: "unknown",
		Region:           "test-region",
		Path:             "test-path",
	}

	_, err := createAWSAuth(config)
	assert.Error(t, err, "createAWSAuth did not fail for unknown signature type")
}
