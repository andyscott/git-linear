#!/usr/bin/env bash

FZF_DEFAULT_COMMAND="git-linear-plumbing" \
fzf --info=inline --layout=reverse --header-lines=1 \
    --read0 --delimiter='\t' \
    --with-nth=1,2 \
    --preview-window up:follow \
    --preview 'echo {3}| glow' \
    --bind 'enter:become(echo git checkout -b {2})'