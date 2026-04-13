#!/bin/bash

# Install the required Nvidia CUDA libs and tools for llama-server
# https://github.com/ggml-org/llama.cpp/tree/master/tools/server

# If you are already root, set empty sudo variable: export sudo=""
sudo=${sudo-sudo}

(
set -xe

$sudo pacman -Syu --noconfirm \
                              \
    cmake                     \
    mimja                     \
    ccache                    \
    npm                       \
    go                        \
                              \
    cuda                      \
    cudnn                     \
    nccl                      \
    nvidia-utils              \
    nvidia-container-toolkit  \
    nvidia-cg-toolkit         \
    nvidia-settings           \
                              \
    wget                      \
    git                       \
    screen                    \
    btop                      \
    htop                      \
)

echo "
You may want to reboot:

    $sudo reboot
"
