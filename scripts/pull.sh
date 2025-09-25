#!/bin/sh

set -xe

# optional parameters: repo and git tag
dir=${1:-${BASH_SOURCE[0]%/*}../../llama.cpp}
tag=$2

cd "$dir"

git fetch

[ -n "$tag" ] && 
	git checkout "$tag" || {
	git switch master || git switch main
	git pull
	}

