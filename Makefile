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
	@echo "Windows system detected - no automated setup available."
	@echo "Please install your developer enviroment manually (@see .tool-versions)."
	@echo
	exit 1
endif
ifeq ($(OSFLAG),$(OSX))
	@echo "MacOS system detected - installing the required toolchain via asdf (@see .tool-versions)."
	@echo
	brew install asdf
	asdf plugin add golang || true
	asdf plugin-add rust || true
	asdf plugin add nodejs || true
	asdf plugin add python || true
	asdf plugin add mockery || true
	asdf plugin add golangci-lint || true
	asdf plugin add actionlint || true
	asdf plugin add shellcheck || true
	asdf plugin add k3d || true
	asdf plugin add kubectl || true
	asdf plugin add k9s || true
	asdf plugin add helm || true
	@echo
	asdf install
endif
ifeq ($(OSFLAG),$(LINUX))
	@echo "Linux system detected - please install and use NIX (@see shell.nix)."
	@echo
ifneq ($(CI),true)
	sh <(curl -L https://nixos-nix-install-tests.cachix.org/serve/vij683ly7sl95nnhb67bdjjfabclr85m/install) --daemon --tarball-url-prefix https://nixos-nix-install-tests.cachix.org/serve --nix-extra-conf-file ./nix.conf
endif
endif

.PHONY: nix-container
nix-container:
	docker run -it --rm -v $(shell pwd):/repo -e NIX_USER_CONF_FILES=/repo/nix.conf --workdir /repo nixos/nix:latest /bin/sh

.PHONY: nix-flake-update
nix-flake-update:
	docker run -it --rm -v $(shell pwd):/repo -e NIX_USER_CONF_FILES=/repo/nix.conf --workdir /repo nixos/nix:latest /bin/sh -c "nix flake update"

build_js:
	yarn install --frozen-lockfile

build_contracts: contracts_compile contracts_install

contracts_compile: artifacts_clean
	./scripts/build-contracts.sh

contracts_install: artifacts_cp_gauntlet artifacts_cp_wasmd

artifacts_cp_gauntlet:
	cp -r artifacts/. packages-ts/gauntlet-cosmos-contracts/artifacts/bin

artifacts_cp_wasmd:
	cp -r artifacts/. ops/wasmd/artifacts

artifacts_clean: artifacts_clean_root artifacts_clean_gauntlet artifacts_clean_wasmd

artifacts_clean_root:
	rm -rf artifacts/*

artifacts_clean_gauntlet:
	rm -rf packages-ts/gauntlet-cosmos-contracts/artifacts/bin/*

artifacts_clean_wasmd:
	rm -rf ops/wasmd/artifacts/*

build: build_js build_contracts

# Common build step
build_relay:
	go build -v ./pkg/cosmos/...

# Unit test without race detection
test_relay_unit: build_relay
	go test -v -covermode=atomic ./pkg/cosmos/... -coverpkg=./... -coverprofile=unit_coverage.txt

# Unit test with race detection
test_relay_unit_race: build_relay
	go test -v -covermode=atomic ./pkg/cosmos/... -race -count=10 -coverpkg=./... -coverprofile=race_coverage.txt


# copied over from starknet, replace as needed
.PHONY: build-go
build-go: build-go-relayer build-go-ops build-go-integration-tests

.PHONY: build-go-relayer
build-go-relayer:
	cd pkg/ && go build ./...

.PHONY: build-go-ops
build-go-ops:
	cd ops/ && go build ./...

.PHONY: build-go-integration-tests
build-go-integration-tests:
	cd integration-tests/ && go build ./...

.PHONY: format-go
format-go: format-go-fmt format-go-mod-tidy

.PHONY: format-go-fmt
format-go-fmt:
	cd ./pkg && go fmt ./...
	cd ./ops && go fmt ./...
	cd ./integration-tests && go fmt ./...

.PHONY: format-go-mod-tidy
format-go-mod-tidy:
	go mod tidy
	cd ./ops && go mod tidy
	cd ./integration-tests && go mod tidy

.PHONY: lint-go
lint-go: lint-go-ops lint-go-relayer lint-go-test

.PHONY: lint-go-ops
lint-go-ops:
	cd ./ops && golangci-lint --color=always run

.PHONY: lint-go-relayer
lint-go-relayer:
	cd ./pkg && golangci-lint --color=always run

.PHONY: lint-go-test
lint-go-test:
	cd ./integration-tests && golangci-lint --color=always --exclude=dot-imports run

.PHONY: test-integration-prep
test-integration-prep:
	# add any stuff that we might need here
	make build

.PHONY: test-go
test-go: test-unit-go test-integration-go

.PHONY: test-unit
test-unit: test-unit-go

.PHONY: test-unit-go
test-unit-go:
	cd ./pkg && go test -v ./...
	cd ./pkg && go test -v ./... -race -count=10

.PHONY: test-integration-go
# only runs tests with TestIntegration_* + //go:build integration
test-integration-go:
	cd ./pkg && go test -v ./... -run TestIntegration -tags integration

.PHONY: test-integration-smoke
test-integration-smoke: test-integration-prep
	cd integration-tests/ && \
		go test --timeout=2h -v ./smoke

# CI Already has already ran test-integration-prep
.PHONY: test-integration-smoke-ci
test-integration-smoke-ci:
	cd integration-tests/ && \
		go test --timeout=2h -v -count=1 -json ./smoke 2>&1 | tee /tmp/gotest.log | gotestfmt

.PHONY: test-integration-remote-runner
test-integration-remote-runner:
	cd integration-tests/ && \
		./"$(suite)".test -test.v -test.count 1 $(args) -test.run ^$(test_name)$

.PHONY: test-integration-soak
test-integration-soak: test-integration-prep
	cd integration-tests/ && \
		go test --timeout=1h -v ./soak

# CI Already has already ran test-integration-prep
.PHONY: test-integration-soak-ci
test-integration-soak-ci:
	cd integration-tests/ && \
		go test --timeout=1h -v -count=1 -json ./soak 2>&1 | tee /tmp/gotest.log | gotestfmt
