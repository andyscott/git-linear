{ stdenv
, callPackage
, go
, lib
, buildGoApplication
, makeWrapper
, coreutils
, fzf
, git
, bash
}:

buildGoApplication {
  pname = "git-linear";
  version = "dev";
  src = ./.;

  inherit go;
  modules = ./gomod2nix.toml;

  nativeBuildInputs = [ makeWrapper ];

  postFixup = ''
    wrapProgram $out/bin/git-linear \
      --prefix PATH ${lib.makeBinPath [
        git
        fzf
        bash # used for sh
        coreutils # used for cat
      ]}
  '';

  meta = {
    description = "Quickly manage git branches for your linear tickets";
    homepage = "https://github.com/andyscott/git-linear";
    license = lib.licenses.mit;
  };
}
