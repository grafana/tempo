{ pkgs ? import <nixpkgs> }:

with pkgs;
buildGoModule rec {
  pname = "mixtool";
  version = "bd0efc3";

  src = fetchFromGitHub {
    owner = "monitoring-mixins";
    repo = pname;
    rev = "${version}";
    sha256 = "1kh2axna553q7lrmgak8l7jlnmbdfkfci240bqa3040pd82j3q1c";
  };

  subPackages = [ "cmd/mixtool" ];
  vendorSha256 = "10wvckrwrc7xs3dng9m6lznsaways2wycxnl9h8jynp4h2cw22ml";

  meta = with lib; {
    description = "Helper for easily working with Jsonnet mixins";
    license = licenses.asl20;
  };
}
