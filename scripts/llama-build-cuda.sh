#!/bin/bash

command -V cmake  || { echo REQUIRED: install cmake  && exit ; }
command -V ninja  || { echo REQUIRED: install ninja  && exit ; }
command -V ccache || { echo REQUIRED: install ccache && exit ; }
#ommand -V cargo  || { echo REQUIRED: install rustup and run: rustup default stable && exit ; }

set -xe

cd ${BASH_SOURCE[0]%/*}  # go to the directory of this script
cd ../../llama.cpp

# build directory
build=build

# Guarantee a clean build by resetting the build directory.
# Fortunately ccache retrieves most of the deleted object files.
rm ${build:-build}/ -rf

flags="$(grep "^flags" -wm1 /proc/cpuinfo) "

# Generate the Ninja files (rather than the Makefile)
cmake -B $build -G Ninja \
  -D BUILD_SHARED_LIBS=OFF \
  -D CMAKE_CUDA_ARCHITECTURES=86 \
  -D CMAKE_EXE_LINKER_FLAGS="-Wl,--allow-shlib-undefined,-flto" \
  -D GGML_AVX=$(        [[ "$flags" == *" avx "*         ]] && echo ON || echo OFF) \
  -D GGML_AVX2=$(       [[ "$flags" == *" avx2 "*        ]] && echo ON || echo OFF) \
  -D GGML_AVX_VNNI=$(   [[ "$flags" == *" vnni "*        ]] && echo ON || echo OFF) \
  -D GGML_AVX512=$(     [[ "$flags" == *" avx512f "*     ]] && echo ON || echo OFF) \
  -D GGML_AVX512_BF16=$([[ "$flags" == *" avx512_bf16 "* ]] && echo ON || echo OFF) \
  -D GGML_AVX512_VBMI=$([[ "$flags" == *" avx512_vbmi "* ]] && echo ON || echo OFF) \
  -D GGML_AVX512_VNNI=$([[ "$flags" == *" avx512_vnni "* ]] && echo ON || echo OFF) \
  -D GGML_BMI2=$(       [[ "$flags" == *" bmi2 "*        ]] && echo ON || echo OFF) \
  -D GGML_SSE42=$(      [[ "$flags" == *" sse4_2 "*      ]] && echo ON || echo OFF) \
  -D GGML_BACKEND_DL=OFF \
  -D GGML_BLAS=OFF \
  -D GGML_CCACHE=ON \
  -D GGML_CPU_ALL_VARIANTS=OFF \
  -D GGML_CUDA_ENABLE_UNIFIED_MEMORY=OFF \
  -D GGML_CUDA_F16=ON \
  -D GGML_CUDA_FA_ALL_QUANTS=ON \
  -D GGML_CUDA=ON \
  -D GGML_LTO=ON \
  -D GGML_NATIVE=ON \
  -D GGML_SCHED_MAX_COPIES=1 \
  -D GGML_STATIC=ON \
  -D LLAMA_BUILD_EXAMPLES=ON \
  -D LLAMA_BUILD_TESTS=OFF \
  -D LLAMA_BUILD_TOOLS=ON \
  -D LLAMA_CURL=ON \
  -D LLAMA_LLGUIDANCE=OFF \
  .

# The build use ninja instead of make
cmake --build $build --config Release --clean-first --target llama-server # llama-gguf
