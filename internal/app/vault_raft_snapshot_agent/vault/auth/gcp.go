package auth

import (
	"github.com/hashicorp/vault/api/auth/gcp"
)

type GCPAuthConfig struct {
	Path                string `default:"gcp"`
	Role                string `validate:"required_if=Empty false"`
	ServiceAccountEmail string
	Empty               bool
}

func createGCPAuth(config GCPAuthConfig) (authMethod, error) {
	var loginOpts = []gcp.LoginOption{gcp.WithMountPath(config.Path)}

	if config.ServiceAccountEmail != "" {
		loginOpts = append(loginOpts, gcp.WithIAMAuth(config.ServiceAccountEmail))
	} else {
		loginOpts = append(loginOpts, gcp.WithGCEAuth())
	}

	auth, err := gcp.NewGCPAuth(config.Role, loginOpts...)
	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
