#!/bin/sh

set -xe

# go to the directory of this script
cd ${BASH_SOURCE[0]%/*}

./pull.sh ${BASH_SOURCE[0]%/*}/../../llama.cpp "$@"
./llama-build-cuda.sh
