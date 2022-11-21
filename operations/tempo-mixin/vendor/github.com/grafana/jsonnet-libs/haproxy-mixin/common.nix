{ pkgs ? import <nixpkgs> }:

with pkgs;
{
  # devTools are packages specifically for development environments.
  devTools = [ docker docker-compose ];
  # buildTools are packages needed for dev and CI builds.
  buildTools = [
    bash
    coreutils
    cue
    drone-cli
    findutils
    git
    gnumake
    gnutar
    jsonnet
    jsonnet-bundler
    mixtool
  ];
}
