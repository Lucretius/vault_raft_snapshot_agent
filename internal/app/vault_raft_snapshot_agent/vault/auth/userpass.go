package auth

import (
	"github.com/hashicorp/vault/api/auth/userpass"
)

type UserPassAuthConfig struct {
	Path     string `default:"userpass"`
	Username string `validate:"required_if=Empty false"`
	Password string `validate:"required_if=Empty false"`
	Empty    bool
}

func createUserPassAuth(config UserPassAuthConfig) (authMethod, error) {
	auth, err := userpass.NewUserpassAuth(
		config.Username,
		&userpass.Password{FromString: config.Password},
		userpass.WithMountPath(config.Path),
	)

	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
