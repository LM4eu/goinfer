#!/bin/bash

command -V cmake  || { echo REQUIRED: install cmake  && exit ; }
command -V ninja  || { echo REQUIRED: install ninja  && exit ; }
command -V cargo  || { echo REQUIRED: install rustup and run: rustup default stable && exit ; }
command -V ccache || { echo REQUIRED: install ccache && exit ; }

cd ${0%/*}  # go to the directory of this script
cd ../../llama.cpp

# Generate the Ninja files (rather than the Makefile)
cmake -B build -G Ninja \
  -D CMAKE_CUDA_ARCHITECTURES=86 \
  -D CMAKE_CUDA_HOST_COMPILER=/usr/bin/g++-14 \
  -D CMAKE_EXE_LINKER_FLAGS="-Wl,--allow-shlib-undefined,-flto" \
  -D GGML_BACKEND_DL=OFF \
  -D GGML_CCACHE=ON \
  -D GGML_CPU_ALL_VARIANTS=OFF \
  -D GGML_CUDA_ENABLE_UNIFIED_MEMORY=ON \
  -D GGML_CUDA_F16=ON \
  -D GGML_CUDA_FA_ALL_QUANTS=ON \
  -D GGML_CUDA=ON \
  -D GGML_LTO=ON \
  -D GGML_NATIVE=ON \
  -D GGML_STATIC=ON \
  -D BUILD_SHARED_LIBS=OFF \
  -D LLAMA_BUILD_EXAMPLES=OFF \
  -D LLAMA_BUILD_TESTS=OFF \
  -D LLAMA_BUILD_TOOLS=ON \
  -D LLAMA_CURL=ON \
  -D LLAMA_LLGUIDANCE=ON \
  .

# The build use ninja instead of make
cmake --build build --config Release --target llama-server
