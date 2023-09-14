package vault

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault/auth"
)

type VaultClientConfig struct {
	Url      string `default:"http://127.0.0.1:8200" validate:"required,http_url"`
	Timeout	 time.Duration `default:"60s"`
	Insecure bool
	Auth     auth.AuthConfig
}

type VaultClientAPI interface {
	TakeSnapshot(ctx context.Context, writer io.Writer) error
	IsLeader() (bool, error)
	AuthAPI() auth.VaultAuthAPI
}

type VaultClient struct {
	Url             string
	api             VaultClientAPI
	auth            auth.Auth
	tokenExpiration time.Time
}

func CreateClient(config VaultClientConfig) (*VaultClient, error) {
	api, err := newVaultAPIImpl(config.Url, config.Insecure, config.Timeout)
	if err != nil {
		return nil, err
	}

	return NewClient(config.Url, api, auth.CreateAuth(config.Auth)), nil
}

func NewClient(address string, api VaultClientAPI, auth auth.Auth) *VaultClient {
	return &VaultClient{address, api, auth, time.Time{}}
}

func (c *VaultClient) TakeSnapshot(ctx context.Context, writer io.Writer) error {
	if err := c.refreshAuth(); err != nil {
		return err
	}

	leader, err := c.api.IsLeader()
	if err != nil {
		return fmt.Errorf("unable to determine leader status for %s: %v", c.Url, err)
	}

	if !leader {
		return fmt.Errorf("%s is not vault-leader-node", c.Url)
	}

	return c.api.TakeSnapshot(ctx, writer)
}

func (c *VaultClient) refreshAuth() error {
	if c.auth != nil && c.tokenExpiration.Before(time.Now()) {
		tokenExpiration, err := c.auth.Refresh(c.api.AuthAPI())
		if err != nil {
			return fmt.Errorf("could not refresh auth: %s", err)
		}
		c.tokenExpiration = tokenExpiration
	}
	return nil
}
