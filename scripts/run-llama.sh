#!/bin/bash

# current directory
cd ${0%/*}

model=${1:-${MODELS_DIR:-../../models}/30b/Qwen3-Coder-30B-A3B-Instruct-UD-Q4_K_XL.gguf}

# --jinja --chat-template-file template.jinja
../../llama.cpp/build/bin/llama-server \
    -fa --log-colors -ngl 99 \
    --no-warmup -c 65536 \
    -m $model
