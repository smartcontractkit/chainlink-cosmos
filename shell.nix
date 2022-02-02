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
    # pkg-config
    # openssl

    (pkgs.callPackage ./terrad.nix {})

    # Golang
    go_1_17
    gopls
    delve
    golangci-lint
    goimports

    # NodeJS + TS
    nodePackages.typescript-language-server
    nodejs-14_x
    (yarn.override { nodejs = nodejs-14_x; })
  ];
  RUST_BACKTRACE = "1";
  LD_LIBRARY_PATH="${stdenv.cc.cc.lib}/lib64:$LD_LIBRARY_PATH";
  GOROOT="${pkgs.go_1_17}/share/go";
  
  # Avoids issues with delve
  CGO_CPPFLAGS="-U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=0";
}
