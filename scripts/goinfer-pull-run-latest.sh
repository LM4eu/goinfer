#!/bin/bash

command -V go  || { echo REQUIRED: install go  && exit ; }
command -V git || { echo REQUIRED: install git && exit ; }

set -xe

cd ${0%/*}  # go to the directory of this script

git -C ../../llama-swap switch main
git -C ../../llama-swap pull

git switch swap
git pull

# Optional
# export GOPROXY=https://proxy.golang.org,direct
# export GODEBUG=toolchaintrace=1
# go env -w GOTOOLCHAIN=auto

cd ../go
go run . "@$"
