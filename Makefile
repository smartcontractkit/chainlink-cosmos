BIN_DIR = bin
export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

download:
	go mod download

install:
	go get github.com/onsi/ginkgo/v2/ginkgo/generators@v2.0.0
	go get github.com/onsi/ginkgo/v2/ginkgo/internal@v2.0.0
	go get github.com/onsi/ginkgo/v2/ginkgo/labels@v2.0.0
	go install github.com/onsi/ginkgo/v2/ginkgo

build_js:
	yarn install --frozen-lockfile && yarn bundle

build_contracts:
	./scripts/build-contracts.sh
	cp -r artifacts packages-ts/gauntlet-terra-contracts/artifacts/bin

build: build_js build_contracts

test_smoke:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo tests/e2e/smoke

test_ocr:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@ocr tests/e2e/smoke

test_gauntlet:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@gauntlet tests/e2e/smoke