#!/bin/sh

set -xe

# go to the directory of this script
cd ${0%/*}

./llama-pull.sh "$@"
./llama-build-cuda.sh
