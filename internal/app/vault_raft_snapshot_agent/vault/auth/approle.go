package auth

import (
	"github.com/hashicorp/vault/api/auth/approle"
)

type AppRoleAuthConfig struct {
	Path     string `default:"approle"`
	RoleId   string `mapstructure:"role" validate:"required_if=Empty false"`
	SecretId string `mapstructure:"secret" validate:"required_if=Empty false"`
	Empty    bool
}

func createAppRoleAuth(config AppRoleAuthConfig) (authMethod, error) {
	auth, err := approle.NewAppRoleAuth(
		config.RoleId,
		&approle.SecretID{FromString: config.SecretId},
		approle.WithMountPath(config.Path),
	)

	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
