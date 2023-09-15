package auth

import (
	"github.com/hashicorp/vault/api/auth/azure"
)

type AzureAuthConfig struct {
	Path     string `default:"azure"`
	Role     string `validate:"required_if=Empty false"`
	Resource string
	Empty    bool
}

func createAzureAuth(config AzureAuthConfig) (authMethod, error) {
	var loginOpts = []azure.LoginOption{azure.WithMountPath(config.Path)}

	if config.Resource != "" {
		loginOpts = append(loginOpts, azure.WithResource(config.Resource))
	}

	auth, err := azure.NewAzureAuth(config.Role, loginOpts...)
	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
