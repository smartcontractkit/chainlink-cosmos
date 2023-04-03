# Build image

FROM golang:1.20-buster as build

# Copy source

RUN mkdir -p /cosmos-monitoring/cmd
COPY ./cmd/monitoring /cosmos-monitoring/cmd/monitoring
COPY ./pkg /cosmos-monitoring/pkg
COPY ./go.mod /cosmos-monitoring/
COPY ./go.sum /cosmos-monitoring/

# Compile binary

WORKDIR /cosmos-monitoring
RUN go build -o ./monitoring ./cmd/monitoring/*.go

RUN go get github.com/CosmWasm/wasmvm

# Production image

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build /cosmos-monitoring/monitoring /monitoring

# Dependency of CosmWasm/wasmd
COPY --from=build /go/pkg/mod/github.com/\!cosm\!wasm/wasmvm@v*/internal/api/libwasmvm.*.so /usr/lib/
RUN chmod 755 /usr/lib/libwasmvm.*.so

# Expose prometheus default port
EXPOSE 9090/tcp

ENTRYPOINT ["/monitoring"]
CMD ["--help"]
