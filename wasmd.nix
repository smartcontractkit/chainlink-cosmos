{ stdenv, pkgs, lib
, fetchFromGitHub
}:

stdenv.mkDerivation rec {
  name = "wasmd-${version}";
  version = "0.3.0";

  src = fetchFromGitHub {
    owner = "CosmWasm";
    repo = "wasmd";
    rev = "0c99c6c64370a86913907b87214f43deef7f2f99";
    sha256 = "0mnim2wwzs1rm8rj00wv304aqzcqvinn5vgyv9cy7kb8jd6fklzx";
  };

  buildInputs = with pkgs; [ gnumake git go_1_20 which openssl cacert ];

  buildPhase = ''
    export HOME=$out
    cd $src && make install
  '';

  installPhase = "ln -s go/bin $out/bin";
}
