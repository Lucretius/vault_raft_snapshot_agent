FROM alpine
WORKDIR /

COPY ./binaries/amd64/vault_raft_snapshot_agent /bin/vault_raft_snapshot_agent
RUN chmod +x /bin/vault_raft_snapshot_agent

VOLUME /etc/vault.d/
ENTRYPOINT ["/bin/vault-raft-snapshot-agent"]
