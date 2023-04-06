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

let

  git-linear-plumbing = buildGoApplication {
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
  };

in

writeShellApplication {
  name = "git-linear";
  runtimeInputs = [
    fzf
    git
    glow
  ];
  text = ''
    FZF_DEFAULT_COMMAND="${git-linear-plumbing}/bin/git-linear-plumbing" \
    fzf --info=inline --layout=reverse --header-lines=1 \
        --read0 --delimiter='\t' \
        --with-nth=1,2 \
        --preview-window up:follow \
        --preview 'echo {3} | glow' \
        --bind 'enter:become(git checkout -b {2})'
  '';
}
