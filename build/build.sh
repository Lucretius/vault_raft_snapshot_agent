#! /bin/bash
set -eu 

BUILD_DIR=${1:-./build}
DIST_DIR=${2:-./dist}
PLATFORM=${4:-linux/amd64}
POST_ACTION=${5:-}

export GO111MODULE=on
export CGO_ENABLED=0
export GOOS=$(dirname "$PLATFORM")
export GOARCH=$(basename "$PLATFORM")
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

cd ${BUILD_DIR}
OUT_DIR="$BUILD_DIR/out"
PLATFORM_OUT_DIR=${OUT_DIR}/${PLATFORM}
mkdir -p ${PLATFORM_OUT_DIR}

VERSION="dev"
if [ -e "$SCRIPT_DIR/VERSION" ]; then
    VERSION=$(cat "$SCRIPT_DIR/VERSION")
fi

echo "Building go source in $(realpath "$BUILD_DIR") for $GOOS/$GOARCH to $(realpath "$PLATFORM_OUT_DIR")..."
go get -v ./...;
go build \
    -a \
    -trimpath \
    -ldflags "-s -w -extldflags '-static' -X 'main.Version=$VERSION'" \
    -tags 'osusergo netgo static_build' \
    -o "${PLATFORM_OUT_DIR}" \
    ./...

while IFS= read -r -d $'\0' binary; do
    DIST_FILE="$(basename $binary)_${GOOS}_${GOARCH}"

    echo "Adding $DIST_FILE to $DIST_DIR..."
    mv $binary "$DIST_DIR/$DIST_FILE"
done < <(find "$OUT_DIR" -type f -print0)

if [ "$POST_ACTION" == "run-agent" ]; then
    AGENT_BINARY="$DIST_DIR/vault-raft-snapshot-agent_${GOOS}_${GOARCH}"
    chmod +x "$AGENT_BINARY"
    exec $AGENT_BINARY
elif [ -n "$POST_ACTION" ]; then
    exec sh -c "$POST_ACTION"
fi
