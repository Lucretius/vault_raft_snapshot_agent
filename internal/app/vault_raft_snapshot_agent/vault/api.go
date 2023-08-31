package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault/auth"
	"github.com/hashicorp/vault/api"
)

func newVaultAPIImpl(address string, insecure bool) (*vaultAPIImpl, error) {
	apiConfig := api.DefaultConfig()
	apiConfig.Address = address

	tlsConfig := &api.TLSConfig{
		Insecure: insecure,
	}

	if err := apiConfig.ConfigureTLS(tlsConfig); err != nil {
		return nil, err
	}

	client, err := api.NewClient(apiConfig)
	if err != nil {
		return nil, err
	}

	return &vaultAPIImpl{
		client,
		&vaultAuthAPIImpl {
			client,
		},
	}, nil
}

type vaultAPIImpl struct {
	client *api.Client
	authAPI *vaultAuthAPIImpl
}

func (impl *vaultAPIImpl) TakeSnapshot(ctx context.Context, writer io.Writer) error {
	return impl.client.Sys().RaftSnapshotWithContext(ctx, writer)
}

func (impl *vaultAPIImpl) IsLeader() (bool, error) {
	leader, err := impl.client.Sys().Leader()
	if err != nil {
		return false, err
	}

	return leader.IsSelf, nil
}

func (impl *vaultAPIImpl) AuthAPI() auth.VaultAuthAPI {
	return impl.authAPI
}

type vaultAuthAPIImpl struct {
	client *api.Client	
}

func (impl *vaultAuthAPIImpl) LoginToBackend(authPath string, credentials map[string]interface{}) (leaseDuration time.Duration, err error) {
	resp, err := impl.client.Logical().Write(path.Clean("auth/"+ authPath +"/login"), credentials)
	if err != nil {
		return 0, err
	}

	impl.client.SetToken(resp.Auth.ClientToken)
	return time.Duration(resp.Auth.LeaseDuration), nil
}

func (impl *vaultAuthAPIImpl) LoginWithToken(token string) (leaseDuration time.Duration, err error) {
	impl.client.SetToken(token)
	info, err := impl.client.Auth().Token().LookupSelf()
	if err != nil {
		impl.client.ClearToken()
		return 0, err
	}

	ttl, err := info.Data["ttl"].(json.Number).Int64()
	if err != nil {
		impl.client.ClearToken()
		return 0, fmt.Errorf("error converting ttl to int: %s", err)
	}

	return time.Duration(ttl), nil
}


