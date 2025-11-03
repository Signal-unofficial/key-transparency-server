ARG GO_VERSION=golang:1.24.0-alpine
ARG ALPINE_VERSION=alpine:latest

# Modifies volume permissions: https://stackoverflow.com/a/73255981
FROM ${ALPINE_VERSION} AS init-volume

WORKDIR /src/
COPY --link --chmod="+x" [ "./docker/init-volume.sh", "./" ]

ENV NEW_UID=1000
ENV NEW_GID=1000

VOLUME [ "/vol/" ]
ENTRYPOINT [ "./init-volume.sh" ]

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

WORKDIR /usr/local/bin/
COPY --from=build-key-generator [ "/src/cmd/generate-keys/generate-keys/", "./" ]

ENTRYPOINT [ "/usr/local/bin/generate-keys" ]

FROM ${GO_VERSION} AS run-tests

WORKDIR /src/
COPY [ "./", "./" ]

ENTRYPOINT [ "go", "test" ]
CMD ["./..."]

FROM ${ALPINE_VERSION} AS run-stress-test

WORKDIR /usr/local/bin/
COPY --from=build-stress-test [ "/src/cmd/kt-stress/kt-stress", "./" ]

ENTRYPOINT [ "/usr/local/bin/kt-stress" ]

FROM ${ALPINE_VERSION} AS run-server

WORKDIR /usr/local/bin/
COPY --from=build-server [ "/src/cmd/kt-server/kt-server", "./" ]

ENTRYPOINT [ "/usr/local/bin/kt-server" ]

FROM ${ALPINE_VERSION} AS run-client

WORKDIR /usr/local/bin/
COPY --from=build-client [ "/src/cmd/kt-client/kt-client", "./" ]

ENTRYPOINT [ "/usr/local/bin/kt-client" ]
