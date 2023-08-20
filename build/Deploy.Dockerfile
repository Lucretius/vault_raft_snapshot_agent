FROM alpine
WORKDIR /

COPY ./binaries/vault_raft_snapshot_agent_linux_amd64 /bin/vault_raft_snapshot_agent
RUN chmod +x /bin/vault_raft_snapshot_agent

VOLUME /etc/vault.d/
ENTRYPOINT ["/bin/vault_raft_snapshot_agent"]
