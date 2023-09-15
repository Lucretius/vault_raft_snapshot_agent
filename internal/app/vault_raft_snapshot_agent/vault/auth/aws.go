package auth

import (
	"fmt"

	"github.com/hashicorp/vault/api/auth/aws"
)

type AWSSignatureType string

const (
	AWS_EC2_PKCS7    AWSSignatureType = "pkcs7"
	AWS_ECS_IDENTITY AWSSignatureType = "identity"
	AWS_EC2_RSA2048  AWSSignatureType = "rsa2048"
)

type AWSAuthConfig struct {
	Path              string `default:"aws"`
	Role              string
	Region            string
	EC2Nonce          string
	EC2SignatureType  AWSSignatureType `default:"pkcs7"`
	IAMServerIDHeader string
	Empty             bool
}

func createAWSAuth(config AWSAuthConfig) (authMethod, error) {
	var loginOpts = []aws.LoginOption{aws.WithMountPath(config.Path)}

	if config.EC2Nonce != "" {
		loginOpts = append(loginOpts, aws.WithNonce(config.EC2Nonce), aws.WithEC2Auth())
		switch config.EC2SignatureType {
		case "":
		case AWS_EC2_PKCS7:
		case AWS_ECS_IDENTITY:
			loginOpts = append(loginOpts, aws.WithIdentitySignature())
		case AWS_EC2_RSA2048:
			loginOpts = append(loginOpts, aws.WithRSA2048Signature())
		default:
			return authMethod{}, fmt.Errorf("unknown signature-type %s", config.EC2SignatureType)
		}
	} else {
		loginOpts = append(loginOpts, aws.WithIAMAuth())
		if config.IAMServerIDHeader != "" {
			loginOpts = append(loginOpts, aws.WithIAMServerIDHeader(config.IAMServerIDHeader))
		}
	}

	if config.Region != "" {
		loginOpts = append(loginOpts, aws.WithRegion(config.Region))
	}

	if config.Role != "" {
		loginOpts = append(loginOpts, aws.WithRole(config.Role))
	}

	auth, err := aws.NewAWSAuth(loginOpts...)
	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
