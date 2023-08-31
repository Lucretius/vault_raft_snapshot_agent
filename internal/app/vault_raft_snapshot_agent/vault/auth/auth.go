package auth

import (
	"time"
)

type AuthConfig struct {
	AppRole    AppRoleAuthConfig    `default:"{\"Empty\": true}"`
	Kubernetes KubernetesAuthConfig `default:"{\"Empty\": true}"`
	Token      string
}

type VaultAuthAPI interface {
	LoginToBackend(path string, credentials map[string]interface{}) (leaseDuration time.Duration, err error)
	LoginWithToken(token string) (leaseDuration time.Duration, err error)
}

type Auth interface {
	Refresh(api VaultAuthAPI) (time.Time, error)
}

func CreateAuth(config AuthConfig) Auth {
	if !config.AppRole.Empty {
		return createAppRoleAuth(config.AppRole)
	} else if !config.Kubernetes.Empty {
		return createKubernetesAuth(config.Kubernetes)
	} else if config.Token != "" {
		return createTokenAuth(config.Token)
	} else {
		return nil
	}
}
