package auth

import (
	"fmt"
	"os"
)

type KubernetesAuthConfig struct {
	Path    string `default:"kubernetes" mapstructure:",omitempty"`
	Role    string `validate:"required_if=Empty false"`
	JWTPath string `default:"/var/run/secrets/kubernetes.io/serviceaccount/token" mapstructure:"," validate:"omitempty,file"`
	Empty   bool
}

func createKubernetesAuth(config KubernetesAuthConfig) authBackend {
	return authBackend{
		name: "Kubernetes",
		path: config.Path,
		credentialsFactory: func() (map[string]interface{}, error) {
			jwt, err := os.ReadFile(config.JWTPath)
			if err != nil {
				return map[string]interface{}{}, fmt.Errorf("unable to read jwt from %s: %s", config.JWTPath, err)
			}

			return map[string]interface{}{
				"role": config.Role,
				"jwt":  string(jwt),
			}, nil
		},
	}
}
