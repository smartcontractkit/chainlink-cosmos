{ stdenv, pkgs, lib
, fetchFromGitHub
}:

stdenv.mkDerivation rec {
  name = "wasmd-${version}";
  version = "0.40.1";

  src = fetchFromGitHub {
    owner = "CosmWasm";
    repo = "wasmd";
    rev = "0c99c6c64370a86913907b87214f43deef7f2f99";
    sha256 = "sha256-KBchSnIETM+HPngqHeajiqLYrhHhXY531C+Q/aSDzl4=";
  };

  buildInputs = with pkgs; [ gnumake git go_1_21 which openssl cacert gcc ];

  buildPhase = ''
    export HOME=$out
    cd $src && make install
  '';

  installPhase = "ln -s go/bin $out/bin";
}
