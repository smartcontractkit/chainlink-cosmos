# Build image

FROM ubuntu:20.04 AS build

# OS dependencies
RUN apt-get update && apt-get install -y wget gcc build-essential git

# Copy source

RUN mkdir -p /terra-monitoring/cmd
COPY ./cmd/monitoring /terra-monitoring/cmd/monitoring
COPY ./pkg /terra-monitoring/pkg
COPY ./go.mod /terra-monitoring/
COPY ./go.sum /terra-monitoring/

# Install golang

RUN wget -c https://dl.google.com/go/go1.18.1.linux-amd64.tar.gz -O - \
  | tar -xz -C /usr/local \
  && mkdir -p /go/src /go/bin
ENV PATH /usr/local/go/bin:$PATH

# Compile binary

WORKDIR /terra-monitoring
RUN go build -o ./monitoring ./cmd/monitoring/*.go

# TODO: Do we need this go get?
RUN go get github.com/CosmWasm/wasmvm

# Production image

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build /terra-monitoring/monitoring /monitoring

# Dependency of terra-money/core - change arch by using `--build-arg LIBWASMVM_ARCH=aarch64`
ARG LIBWASMVM_ARCH=x86_64
COPY --from=1 /go/pkg/mod/github.com/\!cosm\!wasm/wasmvm@v*/api/libwasmvm.${LIBWASMVM_ARCH}.so /usr/lib/libwasmvm.${LIBWASMVM_ARCH}.so
RUN chmod 755 /usr/lib/libwasmvm.${LIBWASMVM_ARCH}.so

# Expose prometheus default port
EXPOSE 9090/tcp

ENTRYPOINT ["/monitoring"]
CMD ["--help"]
