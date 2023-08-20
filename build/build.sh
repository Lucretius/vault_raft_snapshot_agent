#! /bin/bash
set -eu 

BUILD_DIR=${1:-./build}
OUT_DIR=${2:-./out}
ARCH=${3:-amd64}
POST_ACTION=${4:-}

export GO111MODULE=on
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=${ARCH}


ARCH_DIR=${OUT_DIR}/${ARCH}
mkdir -p ${ARCH_DIR}
echo "Building go source in $(realpath "$BUILD_DIR") to $(realpath "$ARCH_DIR")..."
cd ${BUILD_DIR}
go get -v ./...;
go build \
    -a \
    -trimpath \
    -ldflags "-s -w -extldflags '-static'" \
    -tags 'osusergo netgo static_build' \
    -o "${ARCH_DIR}" \
    ./...

if [ "$POST_ACTION" == "run" ]; then
    chmod +x $ARCH_DIR/vault-raft-snapshot-agent
    exec $ARCH_DIR/vault-raft-snapshot-agent
elif [ -n "$POST_ACTION" ]; then
    exec sh -c "$POST_ACTION"
fi
