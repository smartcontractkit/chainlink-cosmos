{ stdenv, pkgs, lib }:

let
  goPkg = if pkgs ? go_1_25 then pkgs.go_1_25 else pkgs.go;
  nodejsPkg = if pkgs ? nodejs_18 then pkgs.nodejs_18 else if builtins.hasAttr "nodejs-18_x" pkgs then pkgs."nodejs-18_x" else pkgs.nodejs;
in
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
    goPkg
    gopls
    delve
    golangci-lint
    gotools
    
    docker-client
    libiconv

    # needed for test
    (if pkgs ? k3d then k3d else kube3d)
    kubectl
    k9s
    kubernetes-helm

    which
    git
    gnumake
    (pkgs.callPackage ./wasmd.nix {})

    # NodeJS + TS
    nodejsPkg
    (yarn.override { nodejs = nodejsPkg; })
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

  # Avoids issues with delve
  CGO_CPPFLAGS="-U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=0";

  HELM_REPOSITORY_CONFIG=./helm-repositories.yaml;
  postShellHook = ''
    go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
    helm repo update
  '';
}
