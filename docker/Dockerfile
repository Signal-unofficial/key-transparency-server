ARG GO_VERSION=golang:1.24.0-alpine
ARG ALPINE_VERSION=alpine:latest

FROM ${GO_VERSION} AS build

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-server/
RUN go build

FROM ${ALPINE_VERSION} AS run

COPY --from=build [ "/src/cmd/kt-server/kt-server", "/bin/" ]

ENTRYPOINT [ "/bin/kt-server" ]
