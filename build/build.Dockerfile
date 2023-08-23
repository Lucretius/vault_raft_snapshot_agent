# Image with go build environment
ARG go_version=1.18
FROM --platform=$TARGETPLATFORM golang:$go_version AS builder

COPY ./build/build.sh /bin/build.sh
RUN chmod +x /bin/build.sh

VOLUME /build
VOLUME /dist
VOLUME /etc/vault.d/

ENV BUILDPLATFORM=$TARGETPLATFORM
ENTRYPOINT ["/bin/build.sh", "--build-dir", "/build", "--dist-dir", "/dist"]
