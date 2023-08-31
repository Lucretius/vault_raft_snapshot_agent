# Image with go build environment
ARG go_version=1.21
FROM --platform=$TARGETPLATFORM golang:$go_version AS builder

COPY ./build/build.sh /bin/build.sh
RUN chmod +x /bin/build.sh

VOLUME /build
VOLUME /work
VOLUME /etc/vault.d/

ENV BUILDPLATFORM=$TARGETPLATFORM
ENV GOPATH=/work/go
ENTRYPOINT ["/bin/build.sh", "--build-dir", "/build", "--dist-dir", "/dist", "--work-dir", "/work"]
