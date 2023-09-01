package auth

type AppRoleAuthConfig struct {
	Path     string `default:"approle"`
	RoleId   string `mapstructure:"id" validate:"required_if=Empty false"`
	SecretId string `mapstructure:"secret" validate:"required_if=Empty false"`
	Empty    bool	
}

func createAppRoleAuth(config AppRoleAuthConfig) authBackend {
	return authBackend{
		name: "AppRole",
		path: config.Path,
		credentialsFactory: func() (map[string]interface{}, error) {
			return map[string]interface{}{
				"role_id":   config.RoleId,
				"secret_id": config.SecretId,
			}, nil
		},
	}
}
