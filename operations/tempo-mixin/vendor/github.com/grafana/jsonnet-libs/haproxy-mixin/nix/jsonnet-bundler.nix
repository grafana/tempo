{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "jsonnet-bundler";
  version = "0.4.0";

  src = fetchFromGitHub {
    owner = pname;
    repo = pname;
    rev = "v${version}";
    sha256 = "0pk6nf8r0wy7lnsnzyjd3vgq4b2kb3zl0xxn01ahpaqgmwpzajlk";
  };

  subPackages = [ "cmd/jb" ];
  vendorSha256 = null;

  meta = with lib; {
    description = "A Jsonnet package manager";
    license = licenses.asl20;
  };
}
