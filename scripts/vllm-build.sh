#!/bin/bash

# current directory
cd ${BASH_SOURCE[0]%/*}
cd ../vllm

export CUDA_HOME=/opt/cuda

# Generate the Ninja files (rather than the Makefile)
cmake -B build -G Ninja \
  -DCMAKE_CUDA_ARCHITECTURES=86 \
  -DCMAKE_CUDA_HOST_COMPILER=/usr/bin/g++-14 \
  -DBUILD_SHARED_LIBS=OFF \
  -DCMAKE_CXX_FLAGS=-march=native \
  -DVLLM_PYTHON_EXECUTABLE=`which python3` \
  .

#   -DVLLM_GPU_LANG=cuda 
#  -DVLLM_TARGET_DEVICE=cuda 

# The build use ninja instead of make
cmake --build build --config Release --target _C
