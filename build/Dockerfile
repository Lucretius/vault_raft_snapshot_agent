FROM alpine
WORKDIR /

COPY ./binaries/$TARGETPLATFORM/vault-raft-snapshot-agent /bin/vault-raft-snapshot-agent
RUN chmod +x /bin/vault-raft-snapshot-agent

VOLUME /etc/vault.d/
ENTRYPOINT ["/bin/vault-raft-snapshot-agent"]
