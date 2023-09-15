package auth

import(
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

type tokenAuth struct {
	token string
}

type tokenAuthAPI interface {
	SetToken(token string)
	LookupToken() (*api.Secret, error)
	ClearToken()
}

func createTokenAuth(token string) tokenAuth {
	return tokenAuth{token}
}

func (auth tokenAuth) Login(ctx context.Context, client *api.Client) (time.Duration, error) {
	return auth.login(tokenAuthImpl{client})
}

func (auth tokenAuth) login(authAPI tokenAuthAPI) (time.Duration, error) {
	authAPI.SetToken(auth.token)
	info, err := authAPI.LookupToken()
	if err != nil {
		authAPI.ClearToken()
		return 0, err
	}

	ttl, err := info.Data["ttl"].(json.Number).Int64()
	if err != nil {
		authAPI.ClearToken()
		return 0, fmt.Errorf("error converting ttl to int: %s", err)
	}

	return time.Duration(ttl), nil

}

type tokenAuthImpl struct {
	client *api.Client
}

func (impl tokenAuthImpl) SetToken(token string) {
	impl.client.SetToken(token)
}

func (impl tokenAuthImpl) LookupToken() (*api.Secret, error) {
	return impl.client.Auth().Token().LookupSelf()
}

func (impl tokenAuthImpl) ClearToken() {
	impl.client.ClearToken()
}