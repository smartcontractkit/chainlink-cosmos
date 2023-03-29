{ stdenv, pkgs, lib
, fetchFromGitHub
}:

stdenv.mkDerivation rec {
  name = "wasmd-${version}";
  version = "0.3.0";

  src = fetchFromGitHub {
    owner = "CosmWasm";
    repo = "wasmd";
    rev = "d48bb69b391edf92228708008dcd5bbe47b83128";
    sha256 = "07nprps0da81krpl2a66qn823ybzghnx55zw7504wpynhgfa0bll";
  };

  buildInputs = with pkgs; [ gnumake git go_1_20 which openssl cacert ];

  buildPhase = ''
    export HOME=$out
    cd $src && make install
  '';

  installPhase = "ln -s go/bin $out/bin";
}
