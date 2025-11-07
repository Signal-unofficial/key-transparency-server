ARG GO_VERSION=golang:1.24.0-alpine
ARG ALPINE_VERSION=alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412
ARG AWS_CLI_VERSION=amazon/aws-cli@sha256:065e642839c546a21ba63ad184a44c510a8ef1130bbe8e2376d819145b5ea039

# Modifies volume permissions:
# https://stackoverflow.com/a/73255981
FROM ${ALPINE_VERSION} AS init-volume

WORKDIR /src/
COPY --link --chmod="+x" [ "./docker/init-volume.sh", "./" ]
ENV PATH=${PATH}:/src/

ENV NEW_UID=1000
ENV NEW_GID=1000

VOLUME [ "/vol/" ]
ENTRYPOINT [ "init-volume.sh" ]

# Initializes DynamoDB tables
FROM ${AWS_CLI_VERSION} AS init-tables

WORKDIR /src/
COPY --link --chmod="+x" [ "./docker/init-tables.sh", "./" ]
ENV PATH=${PATH}:/src/

WORKDIR /aws/
ENTRYPOINT [ "init-tables.sh" ]

# Builds the key generator
FROM ${GO_VERSION} AS build-key-generator

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/generate-keys/
RUN go build

# Builds the stress test
FROM ${GO_VERSION} AS build-stress-test

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-stress/
RUN go build

# Builds the server
FROM ${GO_VERSION} AS build-server

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-server/
RUN go build

# Builds the client
FROM ${GO_VERSION} AS build-client

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-client/
RUN go build

# Runs the key generator
FROM ${ALPINE_VERSION} AS generate-keys

WORKDIR /src/
COPY --from=build-key-generator [ "/src/cmd/generate-keys/generate-keys/", "./" ]
ENV PATH=${PATH}:/src/

ENTRYPOINT [ "generate-keys" ]

# Builds and runs the module tests
FROM ${GO_VERSION} AS run-tests

WORKDIR /src/
COPY --link [ "./", "./" ]

ENTRYPOINT [ "go", "test" ]
CMD [ "./..." ]

# Runs the stress test
FROM ${ALPINE_VERSION} AS run-stress-test

WORKDIR /src/
COPY --from=build-stress-test [ "/src/cmd/kt-stress/kt-stress", "./" ]
ENV PATH=${PATH}:/src/

ENTRYPOINT [ "kt-stress" ]

# Runs the server
FROM ${ALPINE_VERSION} AS run-server

WORKDIR /src/
COPY --from=build-server [ "/src/cmd/kt-server/kt-server", "./" ]
ENV PATH=${PATH}:/src/

ENTRYPOINT [ "kt-server" ]

# Runs the client
FROM ${ALPINE_VERSION} AS run-client

WORKDIR /src/
COPY --from=build-client [ "/src/cmd/kt-client/kt-client", "./" ]
ENV PATH=${PATH}:/src/

ENTRYPOINT [ "kt-client" ]
