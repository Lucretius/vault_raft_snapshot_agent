#! /bin/bash
set -eu 

VALID_ARGS=$(getopt -n $(basename $0) -o b:d:p:tv:w: --long build-dir:,dist-dir:,platform:,test,version:,work-dir: -- "$@")
if [[ $? -ne 0 ]]; then
    exit 1;
fi

PLATFORM=${BUILDPLATFORM:-linux/amd64}
VERSION="Development"
RUN_TESTS=false
WORK_DIR=""

eval set -- "$VALID_ARGS"
while [ : ]; do
  case "$1" in
    -b | --build-dir)
        BUILD_DIR=$2
        shift 2
        ;;
    -d | --dist-dir)
        DIST_DIR=$2
        shift 2
        ;;
    -p | --platform)
        PLATFORM=$2
        shift 2
        ;;
    -t | --test)
        RUN_TESTS=true
        shift
        ;;
    -v | --version)
        VERSION=$2
        shift 2
        ;;
    -w | --work-dir)
        WORK_DIR=$2
        shift 2
        ;;
    --) shift; 
        break 
        ;;
  esac
done

if [ -z "$WORK_DIR" ]; then
    WORK_DIR=$(realpath "$BUILD_DIR/.build")
fi

echo "build-dir: $BUILD_DIR"
echo "work-dir:  $WORK_DIR"
echo "dist-dir:  $DIST_DIR"

export GO111MODULE=on
export CGO_ENABLED=0
export GOOS=$(dirname "$PLATFORM")
export GOARCH=$(basename "$PLATFORM")
export GOFLAGS="-buildvcs=false"

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

cd ${BUILD_DIR}
OUT_DIR="$WORK_DIR/out"
PLATFORM_OUT_DIR=${OUT_DIR}/${PLATFORM}
mkdir -p ${PLATFORM_OUT_DIR}

echo "Downloading dependencies for $GOOS/$GOARCH..."
go get -v ./...;

echo "Building go source in $(realpath "$BUILD_DIR") for $GOOS/$GOARCH to $(realpath "$PLATFORM_OUT_DIR")..."
go build \
    -a \
    -trimpath \
    -ldflags "-s -w -extldflags '-static' -X 'main.Version=$VERSION' -X 'main.Platform=$PLATFORM'" \
    -tags 'osusergo netgo static_build' \
    -o "${PLATFORM_OUT_DIR}" \
    ./...
echo "Build finished."

if [ $RUN_TESTS == "true" ]; then
    echo "Running tests..."
    go test ./...
fi
echo "Tests finished."


while IFS= read -r -d $'\0' binary; do
    DIST_FILE="$(basename $binary)_${GOOS}_${GOARCH}"

    echo "Adding $DIST_FILE to $DIST_DIR..."
    mv $binary "$DIST_DIR/$DIST_FILE"
done < <(find "$OUT_DIR" -type f -print0)

if [ "${1:-}" == "run-agent" ]; then
    shift
    echo "Running agent $DIST_DIR/vault-raft-snapshot-agent_${GOOS}_${GOARCH} $@..."
    AGENT_BINARY="$DIST_DIR/vault-raft-snapshot-agent_${GOOS}_${GOARCH}"
    chmod +x "$AGENT_BINARY"
    exec $AGENT_BINARY $@
elif [ -n "${1:-}" ]; then
    echo "Executing $@..."
    exec sh -c "$@"
fi
