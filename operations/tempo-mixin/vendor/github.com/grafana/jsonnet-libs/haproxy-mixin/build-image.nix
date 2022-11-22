{ pkgs ? import <nixpkgs> }:

let common = import ./common.nix { inherit pkgs; };
in
with pkgs;
dockerTools.buildImage {
  name = "jdbgrafana/haproxy-mixin-build-image";
  created = "now";
  tag = "0.0.5";
  contents = common.buildTools;
}
