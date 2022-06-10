{ stdenv, lib
, fetchurl
#, autoPatchelfHook
}:

stdenv.mkDerivation rec {
  name = "terrad-${version}";
  version = "2.0.1";

  src = fetchurl {
    url = "https://github.com/terra-money/core/releases/download/v${version}/terra_${version}_Linux_x86_64.tar.gz";
    sha256 = "sha256-FBomyS6B8WTNYDk+WnxC8KkYXs7pLEO9UGClESHQxL4=";
  };

  # nativeBuildInputs = [
  #   autoPatchelfHook
  # ];

  sourceRoot = ".";

  installPhase = ''
    install -m755 -D terrad $out/bin/terrad
  '';
}
