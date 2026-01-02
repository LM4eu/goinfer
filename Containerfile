# syntax=docker/dockerfile:1.2
# the above line enables the feature
# "RUN --mount=type=cache,target=..."

# Build:
#
#    docker  build -t goinfer-llama .
#    podman  build -t goinfer-llama .
#    buildah build -t goinfer-llama .
#
# Run:
#
#    docker run --rm -p 8080:8080 -v $PWD/goinfer.ini:/app/goinfer.ini goinfer-llama
#    podman run --rm -p 8080:8080 -v $PWD/goinfer.ini:/app/goinfer.ini goinfer-llama


# Arguments:
#
# pass the argument at build time:
#
#    docker build -t goinfer --build-arg uid=1122 .
#
# uid : to run goinfer as unprivileged (rootless)
ARG uid=1000

# versions to select the Nvidia container image
# see: https://hub.docker.com/r/nvidia/cuda
ARG CUDA_VERSION=12.9
ARG CUDA_PATCH=1
ARG UBUNTU_VERSION=24.04

ARG cuda_full_version=${CUDA_VERSION}.${CUDA_PATCH}
ARG nvidia_dev_image=docker.io/nvidia/cuda:${cuda_full_version}-devel-ubuntu${UBUNTU_VERSION}
ARG nvidia_run_image=docker.io/nvidia/cuda:${cuda_full_version}-runtime-ubuntu${UBUNTU_VERSION}

# --------------------------------------------------------------------
# https://hub.docker.com/_/node
FROM docker.io/node:24-alpine AS infergui-builder

WORKDIR /code

# Download source code
ARG infergui_branch=main
ADD https://github.com/synw/infergui/archive/refs/heads/${infergui_branch}.tar.gz .
RUN tar f ${infergui_branch}.tar.gz -x --strip-components=1
RUN rm    ${infergui_branch}.tar.gz

# build + delivery + clean
RUN set -ex                         ;\
    yarn install --frozen-lockfile  ;\
    yarn cache clean                ;\
    yarn build                      ;\
    mv /code/dist /dist             ;\
    rm -r /code


# --------------------------------------------------------------------
# https://hub.docker.com/_/golang
FROM docker.io/golang:1.25 AS goinfer-builder

ARG uid

# Copy HTTPS root certificates (~200 KB)
RUN mkdir -p                                 /app/etc/ssl/certs
RUN cp -a /etc/ssl/certs/ca-certificates.crt /app/etc/ssl/certs
# Create user & group files
RUN echo "go:x:$uid:$uid::/:" >              /app/etc/passwd
RUN echo "go:x:$uid:"         >              /app/etc/group

WORKDIR /code

COPY go.mod go.sum ./

RUN set -ex          ;\
    go version       ;\
    go mod download

COPY conf    conf
COPY files   files
COPY llama   llama
COPY lm      lm
COPY server  server
COPY state   state
COPY types   types
COPY goinfer.go .

COPY --from=infergui-builder  dist  server/dist

# Reuse Go cache (download...)
ENV GOPATH=/root/go

# Go build flags: "-s -w" removes all debug symbols: https://pkg.go.dev/cmd/link
# GOAMD64=v3 --> https://go.dev/wiki/MinimumRequirements#amd64
RUN --mount=type=cache,target=${GOPATH}        \
    ls -lShA . server/dist                    ;\
    case "$(grep flags -m1 /proc/cpuinfo)" in  \
        *" avx512f "*)  export GOAMD64=v4;;    \
        *" avx2 "*)     export GOAMD64=v3;;    \
        *" sse2 "*)     export GOAMD64=v2;;    \
    esac                                      ;\
    CGO_ENABLED=0                              \
    GOFLAGS="-trimpath -modcacherw"            \
    GOLDFLAGS="-d -s -w -extldflags=-static"   \
    go build -a -v  .

# smoke test
RUN ./goinfer -help

# Copy the eventual dynamic libs
RUN set -ex                                           ;\
    if ldd goinfer ; then                              \
      ldd goinfer |                                    \
      while read lib rest                             ;\
      do                                               \
         find / -path /proc -prune -o -name "$lib" |   \
         while read path                              ;\
         do mkdir -p /app"${path%/*}"      &&          \
            cp -v "$path" /app"${path%/*}" || true    ;\
         done                                         ;\
      done                                            ;\
    fi

RUN cp -v goinfer /app


#------------------------------------------
FROM ${nvidia_dev_image} AS llama-builder

# libgomp1 = GOMP (GCC OpenMP) to execute parallel regions of code across multiple CPU cores
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y --no-install-recommends \
    ccache \
    cmake \
    curl \
    libcurl4-openssl-dev \
    libgomp1 \
    ninja-build \
    ;

# Install the GCC-14 toolchain
RUN apt-get install -y --no-install-recommends \
    cpp-14        \
    g++-14        \
    gcc-14        \
    libc6-dev     \
    libgcc-14-dev \
    ;
RUN update-alternatives --remove-all cpp
RUN update-alternatives --install \
            /usr/bin/cpp                         cpp                           /usr/bin/cpp-14 100                      \
    --slave /usr/bin/gcc                         gcc                           /usr/bin/gcc-14                          \
    --slave /usr/bin/g++                         g++                           /usr/bin/g++-14                          \
    --slave /usr/bin/gcov                        gcov                          /usr/bin/gcov-14                         \
    --slave /usr/bin/gcc-ar                      gcc-ar                        /usr/bin/gcc-ar-14                       \
    --slave /usr/bin/gcc-nm                      gcc-nm                        /usr/bin/gcc-nm-14                       \
    --slave /usr/bin/gcov-dump                   gcov-dump                     /usr/bin/gcov-dump-14                    \
    --slave /usr/bin/gcov-tool                   gcov-tool                     /usr/bin/gcov-tool-14                    \
    --slave /usr/bin/lto-dump                    lto-dump                      /usr/bin/lto-dump-14                     \
    --slave /usr/bin/gcc-ranlib                  gcc-ranlib                    /usr/bin/gcc-ranlib-14                   \
    --slave /usr/bin/x86_64-linux-gnu-cpp        x86_64-linux-gnu-cpp          /usr/bin/x86_64-linux-gnu-cpp-14         \
    --slave /usr/bin/x86_64-linux-gnu-g++        x86_64-linux-gnu-g++          /usr/bin/x86_64-linux-gnu-g++-14         \
    --slave /usr/bin/x86_64-linux-gnu-gcc        x86_64-linux-gnu-gcc          /usr/bin/x86_64-linux-gnu-gcc-14         \
    --slave /usr/bin/x86_64-linux-gnu-gcc-ar     x86_64-linux-gnu-gcc-ar       /usr/bin/x86_64-linux-gnu-gcc-ar-14      \
    --slave /usr/bin/x86_64-linux-gnu-gcc-nm     x86_64-linux-gnu-gcc-nm       /usr/bin/x86_64-linux-gnu-gcc-nm-14      \
    --slave /usr/bin/x86_64-linux-gnu-gcc-ranlib x86_64-linux-gnu-gcc-ranlib   /usr/bin/x86_64-linux-gnu-gcc-ranlib-14  \
    --slave /usr/bin/x86_64-linux-gnu-gcov       x86_64-linux-gnu-gcov         /usr/bin/x86_64-linux-gnu-gcov-14        \
    --slave /usr/bin/x86_64-linux-gnu-gcov-dump  x86_64-linux-gnu-gcov-dump    /usr/bin/x86_64-linux-gnu-gcov-dump-14   \
    --slave /usr/bin/x86_64-linux-gnu-gcov-tool  x86_64-linux-gnu-gcov-tool    /usr/bin/x86_64-linux-gnu-gcov-tool-14   \
    --slave /usr/bin/x86_64-linux-gnu-lto-dump   x86_64-linux-gnu-lto-dump     /usr/bin/x86_64-linux-gnu-lto-dump-14    \
    ;

ARG LLAMA_LLGUIDANCE=ON
# Only if LLAMA_LLGUIDANCE=ON
#   git and cargo required to clone/build github.com/guidance-ai/llguidance
#   guidance-ai/llguidance uses a recent Rust version
ENV RUSTUP_HOME="/root/.rustup"
ENV CARGO_HOME="/root/.cargo"
ENV PATH="${CARGO_HOME}/bin:${PATH}"
RUN if [ x"${LLAMA_LLGUIDANCE}" = x"ON" ] ; then \
        apt-get install -y --no-install-recommends \
            curl \
            git \
        && \
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
            sh -s -- -y --no-modify-path && \
        . $CARGO_HOME/env && \
        rustc --version ; fi
    # Rust smoke test

WORKDIR /prj

# Download source code
# Recent tags: https://github.com/ggml-org/llama.cpp/tags
ARG llama_git_tag=b6131
ADD https://github.com/ggml-org/llama.cpp/archive/refs/tags/${llama_git_tag}.tar.gz .
RUN tar f ${llama_git_tag}.tar.gz -x --strip-components=1
RUN rm    ${llama_git_tag}.tar.gz

# Set 86 for RTX 3090, see https://developer.nvidia.com/cuda-gpus
ARG CMAKE_CUDA_ARCHITECTURES=86
ARG CMAKE_CUDA_HOST_COMPILER=
ARG CMAKE_EXE_LINKER_FLAGS="-Wl,--allow-shlib-undefined"
ARG GGML_BACKEND_DL=OFF
ARG GGML_CCACHE=ON
ARG GGML_CPU_ALL_VARIANTS=OFF
ARG GGML_CUDA_ENABLE_UNIFIED_MEMORY=ON
ARG GGML_CUDA_F16=ON
ARG GGML_CUDA_FA_ALL_QUANTS=ON
ARG GGML_CUDA=ON
ARG GGML_LTO=ON
ARG GGML_NATIVE=ON
ARG GGML_STATIC=ON
ARG BUILD_SHARED_LIBS=OFF
ARG LLAMA_BUILD_EXAMPLES=OFF
ARG LLAMA_BUILD_TESTS=OFF
ARG LLAMA_BUILD_TOOLS=ON
ARG LLAMA_CURL=ON

# Enable ccache for all targets
ENV CC="ccache gcc"
ENV CXX="ccache g++"
ENV CCACHE_DIR="/root/.ccache"

RUN test -n "${CMAKE_CUDA_ARCHITECTURES}" && \
        cmake_cuda_architectures="-D CMAKE_CUDA_ARCHITECTURES=${CMAKE_CUDA_ARCHITECTURES}" || true \
    \
    test -n x"${CMAKE_CUDA_HOST_COMPILER}" && \
        cmake_cuda_host_compiler="-D CMAKE_CUDA_HOST_COMPILER=${CMAKE_CUDA_HOST_COMPILER}" || true \
    \
    test -n x"${CMAKE_EXE_LINKER_FLAGS}" && \
        cmake_exe_linker_flags="-D CMAKE_EXE_LINKER_FLAGS=${CMAKE_EXE_LINKER_FLAGS}" || true \
    \
    test -n x"${GGML_BACKEND_DL}" && \
        ggml_backend_dl="-D GGML_BACKEND_DL=${GGML_BACKEND_DL}" || true \
    \
    test -n x"${GGML_CCACHE}" && \
        ggml_ccache="-D GGML_CCACHE=${GGML_CCACHE}" || true \
    \
    test -n x"${GGML_CPU_ALL_VARIANTS}" && \
        ggml_cpu_all_variants="-D GGML_CPU_ALL_VARIANTS=${GGML_CPU_ALL_VARIANTS}" || true \
    \
    test -n x"${GGML_CUDA_ENABLE_UNIFIED_MEMORY}" && \
        ggml_cuda_enable_unified_memory="-D GGML_CUDA_ENABLE_UNIFIED_MEMORY=${GGML_CUDA_ENABLE_UNIFIED_MEMORY}" || true \
    \
    test -n x"${GGML_CUDA_F16}" && \
        ggml_cuda_f16="-D GGML_CUDA_F16=${GGML_CUDA_F16}" || true \
    \
    ls -l ; \
    \
    cmake -B build -G Ninja \
    ${cmake_cuda_architectures} \
    ${cmake_cuda_host_compiler} \
    ${cmake_exe_linker_flags} \
    ${ggml_backend_dl} \
    ${ggml_ccache} \
    ${ggml_cpu_all_variants} \
    ${ggml_cuda_enable_unified_memory} \
    ${ggml_cuda_f16} \
    -DGGML_CUDA_FA_ALL_QUANTS=${GGML_CUDA_FA_ALL_QUANTS} \
    -DGGML_CUDA=${GGML_CUDA} \
    -DGGML_LTO=${GGML_LTO} \
    -DGGML_NATIVE=${GGML_NATIVE} \
    -DGGML_STATIC=${GGML_STATIC} \
    -DBUILD_SHARED_LIBS=${BUILD_SHARED_LIBS} \
    -DLLAMA_BUILD_EXAMPLES=${LLAMA_BUILD_EXAMPLES} \
    -DLLAMA_BUILD_TESTS=${LLAMA_BUILD_TESTS} \
    -DLLAMA_BUILD_TOOLS=${LLAMA_BUILD_TOOLS} \
    -DLLAMA_CURL=${LLAMA_CURL} \
    -DLLAMA_LLGUIDANCE=${LLAMA_LLGUIDANCE} \
    .
# -D CMAKE_STATIC_LINKER_FLAGS="-static"
# -D CMAKE_SHARED_LINKER_FLAGS=""
# -D CMAKE_CXX_STANDARD_LIBRARIES="-static-libgcc -static-libstdc++"
# -D CMAKE_C_STANDARD_LIBRARIES="-static-libgcc"

RUN --mount=type=cache,target=${CCACHE_DIR} \
    ccache -s && \
    . $CARGO_HOME/env && \
    cmake --build build --config Release --target llama-server && \
    ccache -s

# Prepare the delivery directory
RUN mkdir -pv /app

# First, copy the dynamic libs
RUN set -ex                                           ;\
    ldd build/bin/llama-server |                       \
    while read lib rest                               ;\
    do                                                 \
       find / -path /proc -prune -o -name "$lib" |     \
       while read path                                ;\
       do mkdir -p /app"${path%/*}"      &&            \
          cp -v "$path" /app"${path%/*}" || true      ;\
       done                                           ;\
    done

#    mv usr/lib lib                                    ;\
#    rmdir usr                                         ;\
#    mkdir -p lib64                                    ;\
#    cp /usr/lib64/ld-linux-x86-64.so.2 lib64          ;\
#    ls -lShA /app

# RUN find build -name "*.so" -exec cp -v {} /app + || true
RUN cp -v build/bin/* /app

# llama-server may be run with "--jinja --chat-template-file template.jinja"
COPY scripts/template.jinja /app/


# --------------------------------------------------------------------
# In this tiny image, put only the executable "goinfer",
# eventual lib dependencies and SSL certificates.
# The static web files are within the executable.
# There is nothing else: no shell commands.
FROM scratch AS goinfer-final

# Run as unprivileged
ARG    uid
USER "$uid:$uid"

# Copy the executable, eventual lib dependencies,
# SSL certificates, group and password files
COPY --chown=$uid:$uid --from=goinfer-builder /app .

# Default timezone is UTC
ENV TZ=UTC0

# The default command to run the container
ENTRYPOINT ["./goinfer"]

# Default argument appended to ENTRYPOINT
CMD [""]


# --------------------------------------------------------------------
FROM ${nvidia_run_image} AS final

# Run as unprivileged
ARG    uid
USER "$uid:$uid"

COPY --chown=$uid:$uid --from=goinfer-builder /app /app
COPY --chown=$uid:$uid --from=llama-builder   /app /app

ARG CUDA_VERSION=${CUDA_VERSION}
ENV LD_LIBRARY_PATH=/usr/local/cuda/lib64:/usr/local/cuda-${CUDA_VERSION}/compat

# Default timezone is UTC
ENV TZ=UTC0
ENV LLAMA_ARG_HOST=0.0.0.0

WORKDIR /app

# The default command to run the container
ENTRYPOINT ["/app/goinfer"]

# Default argument appended to ENTRYPOINT
CMD [""]

HEALTHCHECK CMD [ "curl", "-f", "http://localhost:8080/health" ]
