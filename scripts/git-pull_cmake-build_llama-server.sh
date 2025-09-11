#!/bin/sh

set -xe

# go to the directory of this script
cd ${0%/*}

(
	cd ../llama.cpp

	rm build -rf

	git fetch

	[ -n "$1" ] && 
		git checkout "$1" || {
		git switch master
		git pull
		}
)

./build-llama-cuda.sh
