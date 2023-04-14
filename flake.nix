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

        # This function produces a pre-commit installation with fixed tooling/entries
        # on the path. This is tricky because pre-commit is already wrapped and patched
        # by the upstream nix derivation. We don't really want to override that
        # derivation because we want to leverage cached results.
        pre-commit-with-path =
          let upstream = pkgs.pre-commit; in
          path: pkgs.stdenv.mkDerivation rec {
            name = "pre-commit";
            inherit (upstream) buildInputs propagatedBuildInputs;
            nativeBuildInputs = [ pkgs.makeWrapper ];
            phases = [ "installPhase" ];

            installPhase = ''
              mkdir -p $out
              cp -r ${upstream}/* $out/
              for f in $out/lib/*/site-packages/pre_commit/resources/hook-tmpl; do
                # Here we don't substitute in our full path. If we did then we can't
                # update the paths available to an installed hook without requiring
                # users to re-install the hook itself. This isn't a very desireable
                # behavior!
                substituteInPlace $f \
                  --replace ${upstream}/bin/pre-commit pre-commit
              done
              GLOBIGNORE=".:.."
              for f in $out/bin/*; do
                # Here we substitute in our full path because we're patching the
                # wrappers produced by the upstream derivation. We need all
                # references to stay within this new derivation.
                substituteInPlace $f \
                  --replace ${upstream} $out
              done
              chmod +w $out/bin/
              chmod +w $out/bin/pre-commit
              wrapProgram $out/bin/pre-commit \
                --set PATH ${pkgs.lib.makeBinPath path}
            '';
          };
      in
      {
        legacyPackages.default = pkgs;
        devShells = {
          default = pkgs.mkShell {
            packages = with pkgs; [
              just
              go_1_20
              gopls
              gotools
              (pre-commit-with-path [
                git
                golangci-lint
                gomod2nix
                nixpkgs-fmt
                statix
                yamlfmt
              ])
            ];
          };
        };

        packages.default = pkgs.callPackage ./. { };
      });
}
