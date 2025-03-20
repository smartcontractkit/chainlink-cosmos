ARG BASE_IMAGE=local_chainlink

FROM golang:1.23.6 as buildplugins
RUN go version

WORKDIR /build
COPY relayer .
RUN go install ./cmd/chainlink-cosmos

FROM ${BASE_IMAGE}
COPY --from=buildplugins /go/bin/chainlink-cosmos /usr/local/bin/
ENV CL_COSMOS_CMD chainlink-cosmos
