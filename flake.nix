{
  description = "api";

  inputs = {
    nixpkgs = {
      type = "github";
      owner = "NixOS";
      repo = "nixpkgs";
      rev = "0d15ddddc54e04bc34065a9e47024a2c90063f47";
    };
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
    gomod2nix.inputs.utils.follows = "flake-utils";
  };

  outputs = inputs @ { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ inputs.gomod2nix.overlays.default ];
        };
      in
      {
        legacyPackages.default = pkgs;
        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              just
              go_1_20
              gomod2nix
              gopls
              gotools
              golangci-lint
              nixpkgs-fmt
              statix
            ];
          };
        };

        packages.default = pkgs.callPackage ./. { };
      });
}
