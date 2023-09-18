package auth

import (
	"github.com/hashicorp/vault/api/auth/kubernetes"
)

type KubernetesAuthConfig struct {
	Path    string `default:"kubernetes"`
	Role    string `validate:"required_if=Empty false"`
	JWTPath string `default:"/var/run/secrets/kubernetes.io/serviceaccount/token" resolve-path:"" validate:"omitempty,file,required_if=Empty false"`
	Empty   bool
}

func createKubernetesAuth(config KubernetesAuthConfig) (authMethod, error) {
	var loginOpts = []kubernetes.LoginOption{
		kubernetes.WithMountPath(config.Path),
		kubernetes.WithServiceAccountTokenPath(string(config.JWTPath)),
	}

	auth, err := kubernetes.NewKubernetesAuth(config.Role, loginOpts...)
	if err != nil {
		return authMethod{}, err
	}

	return authMethod{auth}, nil
}
