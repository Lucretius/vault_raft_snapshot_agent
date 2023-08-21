FROM alpine
WORKDIR /

ARG BINARY
COPY $BINARY /bin/vault-raft-snapshot-agent
RUN chmod +x /bin/vault-raft-snapshot-agent

VOLUME /etc/vault.d/
ENTRYPOINT ["/bin/vault-raft-snapshot-agent"]
