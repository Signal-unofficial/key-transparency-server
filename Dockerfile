ARG GO_VERSION=golang:1.24.0-alpine
ARG ALPINE_VERSION=alpine:latest

FROM ${GO_VERSION} AS build-key-generator

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/generate-keys/
RUN go build

FROM ${GO_VERSION} AS build-stress-test

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-stress/
RUN go build

FROM ${GO_VERSION} AS build-server

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-server/
RUN go build

FROM ${GO_VERSION} AS build-client

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-client/
RUN go build

FROM ${ALPINE_VERSION} AS generate-keys

COPY --from=build-key-generator [ "/src/cmd/generate-keys/generate-keys/", "/usr/local/bin/" ]

WORKDIR /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/generate-keys" ]

FROM ${GO_VERSION} AS run-tests

WORKDIR /src/
COPY [ "./", "./" ]

ENTRYPOINT [ "go", "test" ]
CMD ["./..."]

FROM ${ALPINE_VERSION} AS run-stress-test

COPY --from=build-stress-test [ "/src/cmd/kt-stress/kt-stress", "/usr/local/bin/" ]

WORKDIR /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/kt-stress" ]

FROM ${ALPINE_VERSION} AS run-server

COPY --from=build-server [ "/src/cmd/kt-server/kt-server", "/usr/local/bin/" ]

WORKDIR /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/kt-server" ]

FROM ${ALPINE_VERSION} AS run-client

COPY --from=build-client [ "/src/cmd/kt-client/kt-client", "/usr/local/bin/" ]

WORKDIR /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/kt-client" ]
