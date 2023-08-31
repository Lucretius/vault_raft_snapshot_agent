package auth

import (
	"fmt"
	"time"
)

type tokenAuth struct {
	token string
}

func createTokenAuth(token string) tokenAuth {
	return tokenAuth{
		token,
	}
}

func (a tokenAuth) Refresh(api VaultAuthAPI) (time.Time, error) {
	leaseDuration, err := api.LoginWithToken(a.token)
	if err != nil {
		return time.Now(), fmt.Errorf("error logging in with token: %s", err)
	}

	return time.Now().Add((time.Second * leaseDuration) / 2), nil
}
