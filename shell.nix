{ stdenv, pkgs, lib }:

pkgs.mkShell {
  nativeBuildInputs = with pkgs; [
     (rust-bin.stable.latest.default.override {
      extensions = ["rust-src"];
      targets = [
        "x86_64-unknown-linux-gnu" # Used on CI
        "wasm32-unknown-unknown"
      ];
    })
    cargo-generate
    cargo-tarpaulin
    gcc
    pkg-config
    openssl
    cacert

    # Golang
    # Keep this golang version in sync with the version in .tool-versions please
    go_1_21
    gopls
    delve
    golangci-lint
    gotools
    
    docker-client
    libiconv

    # needed for test
    kube3d
    kubectl
    k9s
    kubernetes-helm

    which
    git
    gnumake
    (pkgs.callPackage ./wasmd.nix {})

    # NodeJS + TS
    nodejs-18_x
    (yarn.override { nodejs = nodejs-18_x; })
    nodePackages.typescript
    nodePackages.typescript-language-server
    nodePackages.npm

    python3

  ] ++ lib.optionals stdenv.isLinux [
    # ledger specific packages
    libudev-zero
    libusb1
  ];
  RUST_BACKTRACE = "1";
  GOROOT="${pkgs.go_1_21}/share/go";

  # Avoids issues with delve
  CGO_CPPFLAGS="-U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=0";

  HELM_REPOSITORY_CONFIG=./helm-repositories.yaml;
  postShellHook = ''
    go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
    helm repo update
  '';
}
