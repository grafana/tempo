{ pkgs ? import <nixpkgs> }:

with pkgs;
let common = import ./common.nix { inherit pkgs; };
in
mkShell {
  buildInputs = common.buildTools ++ common.devTools;
  shellHook = ''
    export PROMETHEUS_PORT=9091
  '';
}
