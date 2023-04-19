FROM golang:1.20 AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

RUN mkdir /build
WORKDIR /build

COPY . .

RUN go mod download
RUN go build \
        -a \
        -trimpath \
        -ldflags "-s -w -extldflags '-static'" \
        -tags 'osusergo netgo static_build' \
        -o ../vault_raft_snapshot_agent \
        ./main.go

FROM alpine
WORKDIR /
COPY --from=builder /vault_raft_snapshot_agent .
COPY snapshot.json /etc/vault.d/snapshot.json
ENTRYPOINT ["/vault_raft_snapshot_agent"]
