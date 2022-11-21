{
  description = "haproxy-mixin shell development tooling";

  inputs.nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlay =
        (final: prev: {
          jsonnet-bundler = prev.callPackage ./nix/jsonnet-bundler.nix { pkgs = prev; };
          mixtool = prev.callPackage ./nix/mixtool.nix { pkgs = prev; };
        });
    } //
    (flake-utils.lib.eachDefaultSystem
      (system:
        let pkgs = import nixpkgs { inherit system; overlays = [ self.overlay ]; };
        in
        {
          devShell = import ./shell.nix {
            inherit pkgs;
          };
          packages = {
            haproxy-mixin-build-image = import ./build-image.nix { inherit pkgs; };
          };
        }
      ));
}
