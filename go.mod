module github.com/Argelbargel/vault_raft_snapshot_agent

go 1.16

require (
	cloud.google.com/go v0.38.0
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest/adal v0.8.3 // indirect
	github.com/aws/aws-sdk-go v1.30.14
	github.com/hashicorp/vault/api v1.0.4
	go.opencensus.io v0.22.3 // indirect
	golang.org/x/sys v0.0.0-20190523142557-0e01d883c5c5 // indirect
	google.golang.org/api v0.22.0
)

replace github.com/Argelbargel/vault_raft_snapshot_agent/internal/app/vault_raft_snapshot_agent => ./internal/app/vault_raft_snapshot_agent
