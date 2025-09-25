#!/bin/bash

set -xe

gguf=$1
[ -z "$gguf" ] &&
    gguf="$( find ~/.cache/llama.cpp -name '*.gguf' -print -quit )"


# go to the directory of this script
cd ${BASH_SOURCE[0]%/*}

# https://old.reddit.com/r/LocalLLaMA/comments/1kpe33n/speed_up_llamacpp_on_uneven_multigpu_setups_rtx/msxvxk3/

../../llama.cpp/build/bin/llama-gguf "$gguf" r n | 
    awk '/read_0.+size =/ { gsub(/[=,]+/, "", $0); print $6, $4  }' | 
    sort -k1,1rn -k2,2
