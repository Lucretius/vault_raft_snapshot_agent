[![Release](https://img.shields.io/github/v/release/Argelbargel/vault-raft-snapshot-agent)](https://github.com/Argelbargel/vault-raft-snapshot-agent/releases/latest)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/vault-raft-snapshot-agent)](https://artifacthub.io/packages/search?repo=vault-raft-snapshot-agent)

# Vault Raft Snapshot Agent

Vault Raft Snapshot Agent is a Go binary that will take periodic snapshots of a [Vault](https://www.vaultproject.io/) HA cluster using the [integrated raft storage backend](https://developer.hashicorp.com/vault/docs/concepts/integrated-storage). It can store the snapshots locally or upload them to a remote storage backend like AWS S3.


## Running

### Container-Image
You can run the agent with the supplied container-image, e.g. via docker:
```
docker run -v <path to snapshot.json>:/etc/vault.d/snapshot.json" ghcr.io/argelbargel/vault-raft-snapshot-agent:latest
```


### Helm-Chart
If you're running on kubernetes, you can use the provided [Helm-Charts](https://argelbargel.github.io/vault-raft-snapshot-agent-helm/) to install Vault Raft Snapshot Agent into your cluster.


### systemd-service
The recommended way of running this daemon is using systemctl, since it handles restarts and failure scenarios quite well.  To learn more about systemctl, checkout [this article](https://www.digitalocean.com/community/tutorials/how-to-use-systemctl-to-manage-systemd-services-and-units).  begin, create the following file at `/etc/systemd/system/snapshot.service`:

```
[Unit]
Description="An Open Source Snapshot Service for Raft"
Documentation=https://github.com/Argelbargel/vault-raft-snapshot-agent/
Requires=network-online.target
After=network-online.target
ConditionFileNotEmpty=/etc/vault.d/snapshot.json

[Service]
Type=simple
User=vault
Group=vault
ExecStart=/usr/local/bin/vault-raft-snapshot-agent
ExecReload=/usr/local/bin/vault-raft-snapshot-agent
KillMode=process
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Your configuration is assumed to exist at `/etc/vault.d/snapshot.json` and the actual daemon binary at `/usr/local/bin/vault-raft-snapshot-agent`.

Then just run:

```
sudo systemctl enable snapshot
sudo systemctl start snapshot
```

If your configuration is right and Vault is running on the same host as the agent you will see one of the following:

`Not running on leader node, skipping.` or `Successfully created <type> snapshot to <location>`, depending on if the daemon runs on the leader's host or not.


## Configuration

Vault Raft Snaphot Agent looks for it's configuration-file in `/etc/vault.d/` or the current working directory by default. It uses [viper](https://github.com/spf13/viper) as configuration-backend so you can write your configuration in either json, yaml or toml.
You an use `vault-raft-snapshot-agent --config <config-file>` to use a specific configuration file.


#### Example configuration (yaml)
```
vault:
  # Url of the (leading) vault-server
  url: http://vault-server:8200
  auth:
    # configures kubernetes auth
    kubernetes:
      role: "test-role"
snapshots:
  # configures how often snapshots are made, default 1h
  frequency: "4h"
  # configures how many snapshots are retained, default 0
  retain: 10
uploaders:
  # configures local storage of snapshots
  local:
    path: /snapshots
```

(for a complete example with all configuration-options see [complete.yaml](./testdata/complete.yaml))


### Environment variables
Vault Raft Snapshot Agent supports configuration with environment variables. For some common options there are shortcuts defined:
- `VAULT_ADDR` configures the url to the vault-server (same as `vault.url`)
- `AWS_ACCESS_KEY_ID` configures the access key for the AWS uploader (same as `uploaders.aws.credentials.key`) and AWS EC2 authentication
- `AWS_SECRET_ACCESS_KEY` configures the access secret for the AWS uploader (same as `uploaders.aws.credentials.secret`) and AWS EC2 authentication
- `AWS_SESSION_TOKEN` configures the session-token for AWS EC2 authentication
- `AWS_SHARED_CREDENTIALS_FILE` configures AWS EC2 authentication from a file


Any other option can be set by prefixing `VRSA_` to the uppercased path to the key and replacing `.` with `_`. For example `VRSA_SNAPSHOTS_FREQUENCY=<value>` configures the snapshot-frequency and `VRSA_VAULT_AUTH_TOKEN=<value>` configures the token authentication for vault.

_Options specified via environment-variables take precedence before the values specified in the configuration file!_

### Vault configuration
```
vault:
  url: <http(s)-url to vault-server>
  insecure: <true|false>
  timeout: <duration>
```

- `url` *(default: https://127.0.0.1:8200)* - specifies the url of the vault-server. 
  
  **The URL should point be the cluster-leader, otherwise no snapshots get taken until the server the url points to is elected leader!**  When running Vault on Kubernetes installed by the [default helm-chart](https://developer.hashicorp.com/vault/docs/platform/k8s/helm), this should be `http(s)://vault-active.<vault-namespace>.svc.cluster.local:<vault-server service-port>`. 
  
  You can alternatively specify the url with the environment-variable `VAULT_ADDR`


- `insecure` *(default: false)* - specifies whether insecure https connections are allowed or not. Set to `true` when you use self-signed certificates
- `timeout` *(default: 60s)* - timeout for the vault-http-client (see https://golang.org/pkg/time/#ParseDuration for a full list of valid time units); increase for large raft databases (and increase `snapshots.timeout` accordingly!)


### Vault authentication
To allow Vault Raft Snapshot Agent to take snapshots, you must add a policy that allows read-access to the snapshot-apis. This involves the following:

1. `vault login` with an admin user.
2. Create the following policy `vault policy write snapshots ./my_policies/snapshots.hcl` where `snapshots.hcl` is:

```hcl
path "/sys/storage/raft/snapshot"
{
  capabilities = ["read"]
}
```

The above policy is the minimum required policy to be able to generate snapshots. This policy must be associated with the app- or kubernetes-role you specify in you're configuration (see below).

Only one of the following authentication options should be specified. If multiple options are specified *one* of them is used with the following priority: `approle`, `kubernetes`, `token`. If no option is specified, Vault Raft Snapshot Agent tries to access vault unauthenticated (which should fail outside of test- or develop-environments)

When using App-Role or Kubernetes authentication Vault Raft Snapshot Agent automatically renews the authentication when it expires.


#### AppRole authentication

An AppRole allows the snapshot agent to automatically rotate tokens to avoid long-lived credentials. To learn more about AppRole's, see [the Vault docs](https://www.vaultproject.io/docs/auth/approle)

 
##### Minimal configuration
```
vault:
  auth:
    approle:
      role: "<role-id>"
      secret: "<secret-id>"
```

##### Configuration options
- `role` **(required)** - specifies the role_id used to call the Vault API.  See the authentication steps below
- `secret` **(required)** - specifies the secret_id used to call the Vault API
- `path` *(default: approle)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

To allow the App-Role access to the snapshots you should run the following commands on your vault-cluster:
```
vault write auth/<path>/role/snapshot token_policies=snapshots
vault read auth/<path>/role/snapshot/<role-id>
vault write -f auth/<path>/role/snapshot/<secret-id>
```

#### AWS authentication

Uses AWS for authentication (see the [Vault docs](https://developer.hashicorp.com/vault/docs/auth/aws)).

 
##### Minimal configuration
```
vault:
  auth:
    aws:
      role: "<role>"
```

##### Configuration options
- `role` **(required)** - specifies the role used to call the Vault API.  See the authentication steps below
- `ec2Nonce` - enables EC2 authentication and sets the required nonce
- `ec2SignatureType` *(default: pkcs7)* - changes the signature-type for EC2 authentication; valid values are `identity`, `pkcs7` and `rs2048`
- `iamServerIdHeader` - specifies the server-id-header when using IAM authtype
- `region` - specifies the aws region to use
- `path` *(default: aws)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

By default AWS authentication uses the iam authentication type unless `ec2Nonce` is set. The credentials for IAM authentication must be provided via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` or `AWS_SHARED_CREDENTIALS_FILE`). While relative paths normally are resolved relative to the configuration-file, `AWS_SHARED_CREDENTIALS_FILE` must be specified as an absolute path.

To allow the access to the snapshots you should run the following commands on your vault-cluster:
```
# for ec2 authentication
vault write auth/<path>/role/<role> auth_type=ec2 bound_ami_id=<ami-id> policies=snapshots max_ttl=500h

# for iam authentication
vault write auth/<path>/role/<role> auth_type=iam bound_iam_principal_arn=<princial-arn> policies=snapshots max_ttl=500h
```

#### Azure authentication

Authentication using Azure (see [the Vault docs](https://developer.hashicorp.com/vault/docs/auth/azure)).

 
##### Minimal configuration
```
vault:
  auth:
    azure:
      role: "<role-id>"
```

##### Configuration options
- `role` **(required)** - specifies the role used to call the Vault API.  See the authentication steps below
- `resource` - optional azure resource
- `path` *(default: azure)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

To allow the access to the snapshots you should run the following commands on your vault-cluster:
```
vault write auth/<path>/role/<role> \
    policies="snapshots" \
    bound_subscription_ids=<subscription-ids> \
    bound_resource_groups=<resource-group>
```

#### Google Cloud authentication

Authentication using Google Cloud GCE or IAM authentication (see [the Vault docs](https://developer.hashicorp.com/vault/docs/auth/gcp)).

 
##### Minimal configuration
```
vault:
  auth:
    gcp:
      role: "<role>"
```

##### Configuration options
- `role` **(required)** - specifies the role used to call the Vault API.  See the authentication steps below
- `serviceAccountEmail` - activates iam authentication and s specifies the service-account to use
- `path` *(default: gcp)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

By default Google Cloud authentication uses the gce authentication type unless `serviceAccountEmail` is set.

To allow the access to the snapshots you should run the following commands on your vault-cluster:
```
# for iam authentication type
vault write auth/<path>/role/<role> \
    type="iam" \
    policies="snapshots" \
    bound_service_accounts="<service-account-email>"

# for gce authentication type
vault write auth/<path>/role/<role> \
    type="gce" \
    policies="snapshots" \
    bound_projects="<projects>" \
    bound_zones="<zones>" \
    bound_labels="<labels>" \
    bound_service_accounts="<service-acoount-email>"
```


#### Kubernetes authentication
To enable Kubernetes authentication mode, you should follow the steps from [the Vault docs](https://www.vaultproject.io/docs/auth/kubernetes#configuration) and create the appropriate policies and roles.

##### Minimal configuration
```
vault:
  auth:
    kubernetes:
      role: "test"
```

##### Configuration options 
- `role` **(required)** - specifies vault k8s auth role
- `path` *(default: kubernetes)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)
- `jwtPath` *(default: /var/run/secrets/kubernetes.io/serviceaccount/token)* - specifies the path to the file with the JWT-Token for the kubernetes service-account. You may specify the path relative to the location of the configuration file.

To allow kubernetes access to the snapshots you should run the following commands on your vault-cluster:
```
  kubectl -n <your-vault-namespace> exec -it <vault-pod-name> -- vault write auth/<path>/role/<kubernetes.role> bound_service_account_names=*  bound_service_account_namespaces=<namespace of your vault-raft-snapshot-agent-pod> policies=snapshots ttl=24h
```
Depending on your setup you can restrict access to specific service-account-names and/or namespaces.

#### LDAP authentication
Authentication using LDAP (see [the Vault docs](https://developer.hashicorp.com/vault/docs/auth/ldap)).

##### Minimal configuration
```
vault:
  auth:
    ldap:
      role: "test"
```

##### Configuration options 
- `username` **(required)** - the username
- `password` **(required)** - the password
- `path` *(default: ldap)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

To allow access to the snapshots you should run the following commands on your vault-cluster:
```
# allow access for a specific user
vault write auth/<path>/users/<username> policies=snapshot

# allow access based on group
vault write auth/<path>/groups/<group> policies=snapshots
```


#### Token authentication
##### Minimal configuration
```
vault:
  auth:
    token: <token>
```

##### Configuration options
- `token` **(required)** - specifies the token used to login


#### User and Password authentication
Authentication using username and password (see [the Vault docs](https://developer.hashicorp.com/vault/docs/auth/userpass)).

##### Minimal configuration
```
vault:
  auth:
    userpass:
      username: "<username>"
      password: "<password>"
```

##### Configuration options 
- `username` **(required)** - the username
- `password` **(required)** - the password
- `path` *(default: userpass)* - specifies the backend-name used to select the login-endpoint (`auth/<path>/login`)

To allow access to the snapshots you should run the following commands on your vault-cluster:

```
vault write auth/<path>/users/<username> \
    password=<password> \
    policies=snapshots
```


### Snapshot configuration
```
snapshots:
  frequency: <duration>
  timeout: <duration>
  retain: <int>
  namePrefix: <prefix>
  nameSuffix: <suffix>
  timestampFormat: <format>
```

- `frequency` *(default: 1h)* - how often to run the snapshot agent.  Examples: `30s`, `1h`.  See https://golang.org/pkg/time/#ParseDuration for a full list of valid time units
- `retain` *(default: 0)*  -the number of snaphots to retain. For example, if you set `retain: 2`, the two most recent snapshots will be kept in storage. `0` means all snapshots will be retained
- `timeout` *(default: 60s)* - timeout for creating snapshots. Examples: `30s`, `1h`. See https://golang.org/pkg/time/#ParseDuration for a full list of valid time units
- `namePrefix` *(default: raft-snapshot-)* - prefix of the uploaded snapshots 
- `nameSuffix` *(default: .snap)* - suffix/extension of the uploaded snapshots
- `timestampFormat` *(default: 2006-01-02T15-04-05Z-0700)* - timestamp-format for the uploaded snapshots' timestamp, must be valid layout string for [go's time.Format](https://pkg.go.dev/time#Time.Format) - you can test your layout-string at the [Go Playground](https://go.dev/play/p/PxX7LmcPha0).
   
The name of the snapshots is created by concatenating `namePrefix`, the timestamp formatted according to `timestampFormat` and `nameSuffix`, e.g. the defaults would generate  `raft-snapshot-2023-09-01T15-30-00Z+0200.snap` for a snapshot taken at 15:30:00 on 09/01/2023 when the timezone is CEST (GMT + 2h).


### Uploader configuration
```
uploaders:
  # you can configure any of these options (exactly once)
  aws:
    bucket: <bucket>
    credentials:
      key: <key>
      secret: <secret>
  azure:
    accountName: <name>
    accountKey: <key>
    container: <container>
    cloudDomain: <domain>
  google:
    bucket: <bucket>
  local:
    path: <path>
```

Note that if you specify more than one storage option, *all* options will be written to.  For example, specifying `local` and `aws` will write to both locations. Each options can be specified exactly once - thus is is currently not possible to e.g. upload to multiple aws regions by specifying multiple `aws`-entries.


#### AWS S3 Upload
- `bucket` **(required)** - bucket to store snapshots in (required for AWS writes to work)
- `region` *(default: "")* - S3 region if it is required 
- `keyPrefix` *(default: "")* - prefix to store s3 snapshots in.  Defaults to empty string
- `endpoint` *(default: "")* - S3 compatible storage endpoint (ex: http://127.0.0.1:9000)
- `useServerSideEncryption` *(default: false)* -  Encryption is **off** by default. Set to true to turn on AWS' AES256 encryption. Support for AWS KMS keys is not currently supported.
- `forcePathStyle` *(default: false)* - Needed if your S3 Compatible storage support only path-style or you would like to use S3's FIPS Endpoint.


##### AWS authentication
```
uploaders:
  aws:
    credentials:
      key: <key>
      secret: <secret>
```
- `key` **(required)** - specifies the access key. It's recommended to use the standard `AWS_ACCESS_KEY_ID` env var, though
- `secret` **(required)** - specifies the secret It's recommended to use the standard `AWS_SECRET_ACCESS_KEY` env var, though


#### Azure Storage
- `accountName` **(required)** - the account name of the storage account
- `accountKey` **(required)** - the account key of the storage account
- `containerName` **(required)** - the name of the blob container to write to
- `cloudDomain` *(default: blob.core.windows.net) - domain of the cloud-service to use


#### Google Storage
`bucket` **(required)** - the Google Storage Bucket to write to.  Auth is expected to be default machine credentials.


#### Local Storage
`path` **(required)** - fully qualified path, not including file name, for where the snapshot should be written.  i.e. `/raft/snapshots`


## License
- Source code is licensed under MIT

## Contributors
- Vault Raft Snapshot Agent was originally developed by [@Lucretius](https://github.com/Lucretius/vault_raft_snapshot_agent/)
- This build contains improvements done by [@Boostport](https://github.com/Boostport/vault_raft_snapshot_agent/)
- support for additional authentication methods based on code from [@alexeiser](https://github.com/Lucretius/vault_raft_snapshot_agent/pull/25)