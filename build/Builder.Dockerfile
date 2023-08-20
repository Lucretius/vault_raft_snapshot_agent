# Image with go build environment
ARG go_version=1.16
FROM golang:$go_version AS builder

ENV GOOS=linux \
    GOARCH=amd64

COPY ./build/build.sh /bin/build.sh
RUN chmod +x /bin/build.sh
VOLUME /src
VOLUME /out
VOLUME /etc/vault.d/

ENTRYPOINT ["/bin/build.sh", "/build", "/out"]
