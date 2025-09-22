#!/bin/sh

set -xe

# go to the directory of this script
cd ${0%/*}

./pull.sh ../llama.cpp "$@"
./llama-build-cuda.sh
