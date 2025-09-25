#!/bin/sh

set -xe

# go to the directory of this script
cd ${0%/*}

./pull.sh ${0%/*}/../../llama.cpp "$@"
./llama-build-cuda.sh
