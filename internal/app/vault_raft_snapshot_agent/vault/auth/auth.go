package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

type AuthConfig struct {
	AppRole    AppRoleAuthConfig    `default:"{\"Empty\": true}"`
	AWS        AWSAuthConfig        `default:"{\"Empty\": true}"`
	Azure      AzureAuthConfig      `default:"{\"Empty\": true}"`
	GCP        GCPAuthConfig        `default:"{\"Empty\": true}"`
	Kubernetes KubernetesAuthConfig `default:"{\"Empty\": true}"`
	LDAP       LDAPAuthConfig       `default:"{\"Empty\": true}"`
	UserPass   UserPassAuthConfig   `default:"{\"Empty\": true}"`
	Token      string
}

type auth[C any] interface {
	Login(ctx context.Context, client C) (time.Duration, error)
}

type authMethod struct {
	delegate api.AuthMethod
}

func CreateVaultAuth(config AuthConfig) (auth[*api.Client], error) {
	if !config.AppRole.Empty {
		return createAppRoleAuth(config.AppRole)
	} else if !config.AWS.Empty {
		return createAWSAuth(config.AWS)
	} else if !config.Azure.Empty {
		return createAzureAuth(config.Azure)
	} else if !config.GCP.Empty {
		return createGCPAuth(config.GCP)
	} else if !config.Kubernetes.Empty {
		return createKubernetesAuth(config.Kubernetes)
	} else if !config.LDAP.Empty {
		return createLDAPAuth(config.LDAP)
	} else if !config.UserPass.Empty {
		return createUserPassAuth(config.UserPass)
	} else if config.Token != "" {
		return createTokenAuth(config.Token), nil
	} else {
		return nil, fmt.Errorf("unknown authenticatin method")
	}
}

func (wrapper authMethod) Login(ctx context.Context, client *api.Client) (time.Duration, error) {
	secret, err := wrapper.delegate.Login(ctx, client)
	if err != nil {
		return time.Duration(0), err
	}

	return time.Duration(secret.LeaseDuration), nil
}
