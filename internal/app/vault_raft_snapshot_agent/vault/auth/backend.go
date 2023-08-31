package auth

import (
	"fmt"
	"path"
	"time"
)

type authBackend struct {
	name               string
	path               string
	credentialsFactory func() (map[string]interface{}, error)
}

func (b authBackend) Refresh(api VaultAuthAPI) (time.Time, error) {
	credentials, err := b.credentialsFactory()
	if err != nil {
		return time.Now(), fmt.Errorf("error creating credentials for auth-backend %s: %s", b.name, err)
	}

	leaseDuration, err := api.LoginToBackend(path.Clean("auth/"+b.path+"/login"), credentials)
	if err != nil {
		return time.Now(), fmt.Errorf("error logging into vault using auth-backend %s: %s", b.name, err)
	}

	return time.Now().Add((time.Second * leaseDuration) / 2), nil
}
