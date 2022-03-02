BIN_DIR = bin
export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

download:
	go mod download

install:
	go get github.com/onsi/ginkgo/v2/ginkgo/generators@v2.1.2
	go get github.com/onsi/ginkgo/v2/ginkgo/internal@v2.1.2
	go get github.com/onsi/ginkgo/v2/ginkgo/labels@v2.1.2
	go install github.com/onsi/ginkgo/v2/ginkgo

build_js:
	yarn install --frozen-lockfile

build_contracts: contracts_compile contracts_install

contracts_compile: artifacts_clean
	./scripts/build-contracts.sh

contracts_install: artifacts_curl_deps artifacts_cp_gauntlet artifacts_cp_terrad

artifacts_curl_deps: artifacts_curl_cw20

artifacts_curl_cw20:
	curl -Lo artifacts/cw20_base.wasm https://github.com/CosmWasm/cw-plus/releases/download/v0.8.0/cw20_base.wasm

artifacts_cp_gauntlet:
	cp -r artifacts/. packages-ts/gauntlet-terra-contracts/artifacts/bin

artifacts_cp_terrad:
	cp -r artifacts/. ops/terrad/artifacts

artifacts_clean: artifacts_clean_root artifacts_clean_gauntlet artifacts_clean_terrad

artifacts_clean_root:
	rm -rf artifacts/*

artifacts_clean_gauntlet:
	rm -rf packages-ts/gauntlet-terra-contracts/artifacts/bin/*

artifacts_clean_terrad:
	rm -rf ops/terrad/artifacts/*

build: build_js build_contracts

test_relay_unit:
	go build -v ./pkg/terra/...
	go test -v ./pkg/terra/...

test_smoke:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo -p -procs=3 tests/e2e/smoke

test_ocr:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@ocr2 tests/e2e/smoke

test_ocr_proxy:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@ocr_proxy tests/e2e/smoke

test_migration:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo tests/e2e/migration

test_gauntlet:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@gauntlet tests/e2e/smoke
