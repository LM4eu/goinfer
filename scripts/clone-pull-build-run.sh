#!/usr/bin/env bash
# Copyright 2025 The contributors of Goinfer.
# SPDX-License-Identifier: MIT

case "${1:-}" in
  -h|--help)
    cat << EOF
Usage:
    ./clone-pull-build-run.sh [goinfer flags]

This script does the complete clone/pull/build of the goinfer dependencies: llama.cpp and llama-swap
Then the script generates the configuration based on the discovered GUFF files.
Finally the script runs goinfer.

This script can used both to:
- prepare the dev environment (git clone dependencies)
- run goinfer during the development cycle (rebuild dependencies only if new commit)

The script defaults to `git pull` (latest commit).
Example specifying the Git tags of llama.cpp and llama-swap:
    tagLC=b6666 tagLS=166 ./clone-pull-build-run.sh

Provided env. vars have precedence. Example:
    export GI_MODELS_DIR=/home/me/models
    ./clone-pull-build-run.sh

Command line flags are passed to goinfer. Example
    ./clone-pull-build-run.sh -debug -no-api-key

Note: the scripts enables CPU optimizations.
EOF
  exit
  ;;
esac


# --- safe bash ---
set -e           # Exit immediately if any command returns a non‑zero status
set -u           # Unset variable => error
set -o pipefail  # Make a pipeline fail if any component command fails
set -o noclobber # Prevent accidental overwriting of files with > redirection
shopt -s inherit_errexit # Also apply restrictions to $(command substitution)

# print the script line number if something goes wrong
set -E
trap 'echo "ERROR status=$? at ${BASH_SOURCE[0]}:$LINENO" >&2 ; exit "$?"' ERR

# --- this script use external tools ---
command -v git    >/dev/null || { echo REQUIRED: install git    && exit 1; }
command -v go     >/dev/null || { echo REQUIRED: install go     && exit 1; }
command -v npm    >/dev/null || { echo REQUIRED: install npm    && exit 1; }
command -v cmake  >/dev/null || { echo REQUIRED: install cmake  && exit 1; }
command -v ninja  >/dev/null || { echo REQUIRED: install ninja  && exit 1; }
command -v ccache >/dev/null || { echo REQUIRED: install ccache && exit 1; }

# clone_checkout_pull sets build=... to trigger the build
clone_checkout_pull() {
  local repo=$1
  local branch=$2
  local tag=$3
  build=clone # the build reason
  [ -d "${repo#*/}" ] && build= || git clone https://github.com/"$repo"
  cd   "${repo#*/}"
  ( set -x ; pwd ; git fetch --prune --tags )
  if [[ -n "$tag" ]]
  then
    build="tag: $tag"
    ( set -x ; git checkout "$tag" )
  else
    ( set -x ; git switch "$branch" )
    local remote="$(git rev-parse "@{upstream}")"
    local local="$( git rev-parse HEAD)"
    if [[ "$remote" != "$local" ]]
    then
      build="commit: $(git log -1 --pretty=format:%f)"
      ( set -x ; git pull --ff-only )
    fi
  fi
  ( set -x ; git status --short )
}

# CPU flags used to build llama.cpp and goinfer
flags="$(grep "^flags" -wm1 /proc/cpuinfo) " # trailing space required

llamaCpp() {
  cd "${BASH_SOURCE[0]%/*}/../.."
  clone_checkout_pull ggml-org/llama.cpp "${branchLC:-master}" "${tagLC:-}"
  [[ -n "$build" ]] || { [[ -f build/bin/llama-server ]] || build="missing build/bin/llama-server" ; }
  [[ -z "$build" ]] || (
    echo "Build llama.cpp because $build"
    rm -rf build/  # this guarantees a clean build, ccache will restore deleted files
    GGML_AVX=$(        [[ "$flags" == *" avx "*         ]] && echo ON || echo OFF)
    GGML_AVX2=$(       [[ "$flags" == *" avx2 "*        ]] && echo ON || echo OFF)
    GGML_AVX_VNNI=$(   [[ "$flags" == *" vnni "*        ]] && echo ON || echo OFF)
    GGML_AVX512=$(     [[ "$flags" == *" avx512f "*     ]] && echo ON || echo OFF)
    GGML_AVX512_BF16=$([[ "$flags" == *" avx512_bf16 "* ]] && echo ON || echo OFF)
    GGML_AVX512_VBMI=$([[ "$flags" == *" avx512_vbmi "* ]] && echo ON || echo OFF)
    GGML_AVX512_VNNI=$([[ "$flags" == *" avx512_vnni "* ]] && echo ON || echo OFF)
    GGML_BMI2=$(       [[ "$flags" == *" bmi2 "*        ]] && echo ON || echo OFF)
    GGML_SSE42=$(      [[ "$flags" == *" sse4_2 "*      ]] && echo ON || echo OFF)
    set -
    pwd
    # generate the Ninja files (faster than Makefile)
    cmake -B build/ -G Ninja                                                                  \
      -D BUILD_SHARED_LIBS=${BUILD_SHARED_LIBS:-OFF}                                          \
      -D CMAKE_BUILD_TYPE=Release                                                             \
      -D CMAKE_CUDA_ARCHITECTURES=${CMAKE_CUDA_ARCHITECTURES:-86}                             \
      -D CMAKE_EXE_LINKER_FLAGS=${CMAKE_EXE_LINKER_FLAGS:-"-Wl,--allow-shlib-undefined,-flto"}\
      -D GGML_AVX=$GGML_AVX                                                                   \
      -D GGML_AVX2=$GGML_AVX2                                                                 \
      -D GGML_AVX_VNNI=$GGML_AVX_VNNI                                                         \
      -D GGML_AVX512=$GGML_AVX512                                                             \
      -D GGML_AVX512_BF16=$GGML_AVX512_BF16                                                   \
      -D GGML_AVX512_VBMI=$GGML_AVX512_VBMI                                                   \
      -D GGML_AVX512_VNNI=$GGML_AVX512_VNNI                                                   \
      -D GGML_BMI2=$GGML_BMI2                                                                 \
      -D GGML_SSE42=$GGML_SSE42                                                               \
      -D GGML_BACKEND_DL=${GGML_BACKEND_DL:-OFF}                                              \
      -D GGML_BLAS=${GGML_BLAS:-OFF}                                                          \
      -D GGML_CCACHE=${GGML_CCACHE:-ON}                                                       \
      -D GGML_CPU_ALL_VARIANTS=${GGML_CPU_ALL_VARIANTS:-OFF}                                  \
      -D GGML_CUDA_ENABLE_UNIFIED_MEMORY=${GGML_CUDA_ENABLE_UNIFIED_MEMORY:-OFF}              \
      -D GGML_CUDA_F16=${GGML_CUDA_F16:-ON}                                                   \
      -D GGML_CUDA_FA_ALL_QUANTS=${GGML_CUDA_FA_ALL_QUANTS:-ON}                               \
      -D GGML_CUDA=${GGML_CUDA:-$(command -v nvcc >/dev/null && echo ON || echo OFF)}         \
      -D GGML_LTO=${GGML_LTO:-ON}                                                             \
      -D GGML_NATIVE=${GGML_NATIVE:-ON}                                                       \
      -D GGML_SCHED_MAX_COPIES=${GGML_SCHED_MAX_COPIES:-1}                                    \
      -D GGML_STATIC=${GGML_STATIC:-ON}                                                       \
      -D LLAMA_BUILD_EXAMPLES=${LLAMA_BUILD_EXAMPLES:-ON}                                     \
      -D LLAMA_BUILD_TESTS=${LLAMA_BUILD_TESTS:-OFF}                                          \
      -D LLAMA_BUILD_TOOLS=${LLAMA_BUILD_TOOLS:-ON}                                           \
      -D LLAMA_CURL=${LLAMA_CURL:-ON}                                                         \
      -D LLAMA_LLGUIDANCE=${LLAMA_LLGUIDANCE:-OFF}                                            \
      .
    cmake --build build/ --config Release --clean-first --target llama-server # llama-gguf
  )
}


llamaSwap(){
  cd "${BASH_SOURCE[0]%/*}/../.."
  clone_checkout_pull LM4eu/llama-swap "${branchLS:-main}" "${tagLS:-}"
  [[ -n "$build" ]] || { [[ -f proxy/ui_dist/index.html ]] || build="missing proxy/ui_dist/index.html" ; }
  [[ -z "$build" ]] || (
    echo "Build llama-swap because $build"
    # we may: rm proxy/ui_dist/
    set -x
    cd ui
    pwd
    npm ci --prefer-offline --no-audit --no-fund --omit=dev
    npm run build
  )
}

GI_MODELS_DIR="${GI_MODELS_DIR:-$(p=;find "$HOME" /mnt -type f -name '*.gguf' -printf '%h\0'|
sort -zu|while IFS= read -rd '' d;do [[ $p && $d == "$p"/* ]] && continue;echo -n "$d:";p=$d;done)}"

export GI_MODELS_DIR=${GI_MODELS_DIR:?GI_MODELS_DIR is empty: Download a model file *.gguf or set GI_MODELS_DIR}

# clone/pull/build llama.cpp if GI_LLAMA_EXE is unset/empty
export GI_LLAMA_EXE="${GI_LLAMA_EXE:-"$(llamaCpp >&2 && cd "${BASH_SOURCE[0]%/*}/../../llama.cpp/build/bin" && pwd )"/llama-server}"

llamaSwap

# --- goinfer ---
cd "${BASH_SOURCE[0]%/*}/../go"
case "$flags" in
    *" avx512f "*)  export GOAMD64=v4;;
    *" avx2 "*)     export GOAMD64=v3;;
    *" sse2 "*)     export GOAMD64=v2;;
esac
set -x
pwd
go build .

# --- generate config ---
./goinfer -gen-main-cfg "$@"
./goinfer -gen-swap-cfg "$@"

# --- run goinfer ---
export GIN_MODE="${GIN_MODE:-release}"
./goinfer "$@"
