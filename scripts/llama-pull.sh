#!/bin/sh

set -xe

# optional parameter: git tag
tag=$1

# go to the directory of this script
cd ${0%/*}

cd ../../llama.cpp

git fetch

[ -n "$tag" ] && 
	git checkout "$tag" || {
	git switch master
	git pull
	}

