{ stdenv
, callPackage
, go
, lib
, buildGoApplication
, writeShellApplication
, fzf
, git
, glow
}:

buildGoApplication {
  pname = "git-linear";
  version = "dev";
  src = ./.;

  inherit go;
  modules = ./gomod2nix.toml;

  meta = {
    description = "Quickly manage git branches for your linear tickets";
    homepage = "https://github.com/andyscott/git-linear";
    license = lib.licenses.mit;
  };
}
