# Image with go build environment
ARG go_version=1.16
FROM golang:$go_version AS builder

COPY ./build/build.sh /bin/build.sh
RUN chmod +x /bin/build.sh

VOLUME /build
VOLUME /dist
VOLUME /etc/vault.d/

ENTRYPOINT ["/bin/build.sh", "/build", "/dist"]
