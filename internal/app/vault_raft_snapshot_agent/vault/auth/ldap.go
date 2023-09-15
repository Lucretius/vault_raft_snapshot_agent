package auth

import (
	"github.com/hashicorp/vault/api/auth/ldap"
)

type LDAPAuthConfig struct {
	Path     string `default:"ldap"`
	Username string `validate:"required_if=Empty false"`
	Password string `validate:"required_if=Empty false"`
	Empty    bool
}

func createLDAPAuth(config LDAPAuthConfig) (authMethod, error) {
	auth, err := ldap.NewLDAPAuth(
		config.Username,
		&ldap.Password{FromString: config.Password},
		ldap.WithMountPath(config.Path),
	)

	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
