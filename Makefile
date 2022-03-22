BIN_DIR = bin
export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

LINUX=LINUX
OSX=OSX
WINDOWS=WIN32
OSFLAG :=
ifeq ($(OS),Windows_NT)
	OSFLAG = $(WINDOWS)
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		OSFLAG = $(LINUX)
	endif
	ifeq ($(UNAME_S),Darwin)
		OSFLAG = $(OSX)
	endif
endif

download:
	go mod download

install:
ifeq ($(OSFLAG),$(WINDOWS))
	echo "If you are running windows and know how to install what is needed, please contribute by adding it here!"
	exit 1
endif
ifeq ($(OSFLAG),$(OSX))
	brew install asdf
	asdf plugin-add nodejs https://github.com/asdf-vm/asdf-nodejs.git || true
	asdf plugin-add rust https://github.com/code-lever/asdf-rust.git || true
	asdf plugin-add golang https://github.com/kennyp/asdf-golang.git || true
	asdf plugin-add ginkgo https://github.com/jimmidyson/asdf-ginkgo.git || true
	asdf plugin-add pulumi || true
	asdf install
endif
ifeq ($(OSFLAG),$(LINUX))
	# install nix
ifneq ($(CI),true)
	sh <(curl -L https://nixos-nix-install-tests.cachix.org/serve/vij683ly7sl95nnhb67bdjjfabclr85m/install) --daemon --tarball-url-prefix https://nixos-nix-install-tests.cachix.org/serve --nix-extra-conf-file ./nix.conf
endif
	go install github.com/onsi/ginkgo/v2/ginkgo@v$(shell cat ./.tool-versions | grep ginkgo | sed -En "s/ginkgo.(.*)/\1/p")
endif

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

test_ocr_soak:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@ocr2-soak tests/e2e/soak

test_ocr_proxy:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@ocr_proxy tests/e2e/smoke

test_migration:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo tests/e2e/migration

test_gauntlet:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo --focus=@gauntlet tests/e2e/smoke

test_chaos:
	SELECTED_NETWORKS=localterra NETWORK_SETTINGS=$(shell pwd)/tests/e2e/networks.yaml ginkgo tests/e2e/chaos
