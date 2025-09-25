#!/bin/bash

command -V git || { echo REQUIRED: install git && exit ; }
command -V go  || { echo REQUIRED: install go  && exit ; }
command -V npm || { echo REQUIRED: install npm && exit ; }

set -xe

cd ${0%/*}  # go to the directory of this script

# --- llama-swap ---
cd ../..
[ -d llama-swap ] || git clone https://github.com/mostlygeek/llama-swap
cd   llama-swap/ui
git pull
npm i
npm run build

# --- infergui ---
cd ../..
[ -d infergui ] || git clone https://github.com/synw/infergui
cd   infergui
git pull
npm i
npm run build

# --- copy /dist from inferui to goinfer ---
rm -rf     ../goinfer/go/infer/dist
cp -r dist ../goinfer/go/infer

# --- goinfer ---
cd ../goinfer/go
git pull

# --- use best perf ---
case "$(grep flags -m1 /proc/cpuinfo)" in
    *" avx512f "*)  export GOAMD64=v4;;
    *" avx2 "*)     export GOAMD64=v3;;
    *" sse2 "*)     export GOAMD64=v2;;
esac

go run . "@$"
