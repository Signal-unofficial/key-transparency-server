ARG GO_VERSION=1.24.0

FROM --platform=linux/amd64 golang:${GO_VERSION}-alpine AS build

COPY [ "./", "/src/" ]
WORKDIR /src/cmd/kt-server/
RUN go build

FROM --platform=linux/amd64 alpine AS run
COPY --from=build [ "/src/cmd/kt-server/kt-server/", "/bin/kt-server/" ]

ENTRYPOINT [ "/bin/kt-server" ]
