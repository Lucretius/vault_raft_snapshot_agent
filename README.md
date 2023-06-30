![Build](https://github.com/Lucretius/vault_raft_snapshot_agent/workflows/Build/badge.svg?branch=master)

# Raft Snapshot Agent

Raft Snapshot Agent is a Go binary that is meant to run alongside every member of a Vault cluster and will take periodic snapshots of the Raft database and write it to the desired location.  It's configuration is meant to somewhat parallel that of the [Consul Snapshot Agent](https://www.consul.io/docs/commands/snapshot/agent.html) so many of the same configuration properties you see there will be present here.

## "High Availability" explained
It works in an "HA" way as follows:
1) Each running daemon checks the IP address of the machine its running on.
2) If this IP address matches that of the leader node, it will be responsible for performing snapshotting.
3) The other binaries simply continue checking, on each snapshot interval, to see if they have become the leader.

In this way, the daemon will always run on the leader Raft node.

Another way to do this, which would allow us to run the snapshot agent anywhere, is to simply have the daemons form their own Raft cluster, but this approach seemed much more cumbersome.

## Running

The recommended way of running this daemon is using systemctl, since it handles restarts and failure scenarios quite well.  To learn more about systemctl, checkout [this article](https://www.digitalocean.com/community/tutorials/how-to-use-systemctl-to-manage-systemd-services-and-units).  begin, create the following file at `/etc/systemd/system/snapshot.service`:

```
[Unit]
Description="An Open Source Snapshot Service for Raft"
Documentation=https://github.com/Lucretius/vault_raft_snapshot_agent/
Requires=network-online.target
After=network-online.target
ConditionFileNotEmpty=/etc/vault.d/snapshot.json

[Service]
Type=simple
User=vault
Group=vault
ExecStart=/usr/local/bin/vault_raft_snapshot_agent
ExecReload=/usr/local/bin/vault_raft_snapshot_agent
KillMode=process
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Your configuration is assumed to exist at `/etc/vault.d/snapshot.json` and the actual daemon binary at `/usr/local/bin/vault_raft_snapshot_agent`.

Then just run:

```
sudo systemctl enable snapshot
sudo systemctl start snapshot
```

If your configuration is right and Vault is running on the same host as the agent you will see one of the following:

`Not running on leader node, skipping.` or `Successfully created <type> snapshot to <location>`, depending on if the daemon runs on the leader's host or not.

## Configuration

`addr` The address of the Vault cluster.  This is used to check the Vault cluster leader IP, as well as generate snapshots. Defaults to "https://127.0.0.1:8200".

`retain` The number of backups to retain.

`frequency` How often to run the snapshot agent.  Examples: `30s`, `1h`.  See https://golang.org/pkg/time/#ParseDuration for a full list of valid time units.

`snapshot_timeout` Timeout for creating snapshots.  Examples: `30s`, `1h`. Default: `60s`. See https://golang.org/pkg/time/#ParseDuration for a full list of valid time units.

### Default authentication mode
`role_id` Specifies the role_id used to call the Vault API.  See the authentication steps below.

`secret_id` Specifies the secret_id used to call the Vault API.

`approle` Specifies the approle name used to login.  Defaults to "approle".

### Kubernetes authentication mode
Incase we're running the application under kubernetes, we can use Vault's Kubernetes Auth
as below. Read more on [kubernetes auth mode](https://www.vaultproject.io/docs/auth/kubernetes)

`vault_auth_method` Set it to "k8s", otherwise, approle will be chosen

`k8s_auth_role` Specifies vault k8s auth role

`k8s_auth_path` Specifies vault k8s auth path

### Token authentication mode
Authenticates with vault using a supplied token.

`vault_auth_method` Set it to "token", otherwise, approle will be chosen

`token` Specifies the vault token

### Storage options

Note that if you specify more than one storage option, *all* options will be written to.  For example, specifying `local_storage` and `aws_storage` will write to both locations.

`local_storage` - Object for writing to a file on disk.

`aws_storage` - Object for writing to an S3 bucket (Support AWS S3 but also S3 Compatible Storage).

`google_storage` - Object for writing to GCS.

`azure_storage` - Object for writing to Azure.

#### Local Storage

`path` - Fully qualified path, not including file name, for where the snapshot should be written.  i.e. /etc/raft/snapshots

#### AWS Storage

`access_key_id` - Recommended to use the standard `AWS_ACCESS_KEY_ID` env var, but its possible to specify this in the config

`secret_access_key` - Recommended to use the standard `SECRET_ACCESS_KEY` env var, but its possible to specify this in the config

`s3_endpoint` - S3 compatible storage endpoint (ex: http://127.0.0.1:9000)

`s3_force_path_style` - Needed if your S3 Compatible storage support only path-style or you would like to use S3's FIPS Endpoint.

`s3_region` - S3 region as is required for programmatic interaction with AWS

`s3_bucket` - bucket to store snapshots in (required for AWS writes to work)

`s3_key_prefix` - Prefix to store s3 snapshots in.  Defaults to nothing.

`s3_server_side_encryption` -  Encryption is **off** by default.  Set to true to turn on AWS' AES256 encryption.  Support for AWS KMS keys is not currently supported.

#### Google Storage

`bucket` - The Google Storage Bucket to write to.  Auth is expected to be default machine credentials.

#### Azure Storage

`account_name` - The account name of the storage account

`account_key` - The account key of the storage account

`container_name` The name of the blob container to write to


## Authentication


### Default authentication mode

You must do some quick initial setup prior to being able to use the Snapshot Agent.  This involves the following:

`vault login` with an admin user.
Create the following policy `vault policy write snapshot ./my_policies/snapshot_policy.hcl`
 where `snapshot_policy.hcl` is:

```hcl
path "/sys/storage/raft/snapshot"
{
  capabilities = ["read"]
}
```

Then run:
```
vault write auth/approle/role/snapshot token_policies="snapshot"
vault read auth/approle/role/snapshot/role-id
vault write -f auth/approle/role/snapshot/secret-id
```

and copy your secret and role ids, and place them into the snapshot file.  The snapshot agent will use them to request client tokens, so that it can interact with your Vault cluster.  The above policy is the minimum required policy to be able to generate snapshots.  The snapshot agent will automatically renew the token when it is going to expire.

The AppRole allows the snapshot agent to automatically rotate tokens to avoid long-lived credentials.

To learn more about AppRole's and why this project chose to use them, see [the Vault docs](https://www.vaultproject.io/docs/auth/approle)


### Kubernetes authentication mode

To Enable Kubernetes authentication mode, we should follow these steps from [the Vault docs](https://www.vaultproject.io/docs/auth/kubernetes#configuration)
