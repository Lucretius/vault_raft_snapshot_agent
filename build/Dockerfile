FROM alpine

LABEL org.opencontainers.image.source=https://github.com/Argelbargel/vault-raft-snapshot-agent
LABEL org.opencontainers.image.description="vault-raft-snapshot-agent ($TARGETPLATFORM)"
LABEL org.opencontainers.image.licenses=MIT

ENTRYPOINT ["/bin/vault-raft-snapshot-agent"]
VOLUME /etc/vault.d/
WORKDIR /

COPY ./binaries/vault-raft-snapshot-agent_$TARGETOS_$TARGETARCH /bin/vault-raft-snapshot-agent
RUN chmod +x /bin/vault-raft-snapshot-agent
