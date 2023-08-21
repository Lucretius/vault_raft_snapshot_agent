#! /bin/bash
set -eu 

BUILD_DIR=${1:-./build}
OUT_DIR=${2:-./out}
PLATFORM=${3:-linux/amd64}
POST_ACTION=${4:-}

export GO111MODULE=on
export CGO_ENABLED=0
export GOOS=$(dirname "$PLATFORM")
export GOARCH=$(basename "$PLATFORM")


PLATFORM_OUT_DIR=${OUT_DIR}/${PLATFORM}
mkdir -p ${PLATFORM_OUT_DIR}
echo "Building go source in $(realpath "$BUILD_DIR") for $GOOS/$GOARCH to $(realpath "$PLATFORM_OUT_DIR")..."
cd ${BUILD_DIR}
go get -v ./...;
go build \
    -a \
    -trimpath \
    -ldflags "-s -w -extldflags '-static'" \
    -tags 'osusergo netgo static_build' \
    -o "${PLATFORM_OUT_DIR}" \
    ./...

if [ "$POST_ACTION" == "run" ]; then
    chmod +x $PLATFORM_OUT_DIR/vault-raft-snapshot-agent
    exec $PLATFORM_OUT_DIR/vault-raft-snapshot-agent
elif [ -n "$POST_ACTION" ]; then
    exec sh -c "$POST_ACTION"
fi
