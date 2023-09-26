# Build:
#
#    docker  build -t goinfer .
#    podman  build -t goinfer .
#    buildah build -t goinfer .
#
# Run:
#
#    docker run --rm -p 5143:5143 --name goinfer goinfer
#    podman run --rm -p 5143:5143 --name goinfer goinfer


ARG uid=5505


# --------------------------------------------------------------------
# https://hub.docker.com/_/node
FROM docker.io/node:20-bookworm AS infergui

WORKDIR /code

# Clone repo + build + clean
RUN set -ex                                                         ;\
    git --version                                                   ;\
    git clone https://github.com/synw/infergui .                    ;\
    ls -lA                                                          ;\
    yarn versions                                                   ;\
    yarn install --frozen-lockfile                                  ;\
    yarn cache clean                                                ;\
    ls -lA                                                          ;\
    yarn build                                                      ;\
    mv /code/dist /dist                                             ;\
    rm -r /code


# --------------------------------------------------------------------
# https://hub.docker.com/_/golang
FROM docker.io/golang:1.21 AS llama

WORKDIR /code

RUN git clone --recurse-submodules https://github.com/go-skynet/go-llama.cpp

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && apt install -y patch cmake && rm -rf /var/lib/apt/lists/*

RUN make -C go-llama.cpp libbinding.a


# --------------------------------------------------------------------
FROM docker.io/golang:1.21 AS goinfer

WORKDIR /code

COPY go.mod go.sum ./

COPY --from=llama  code/go-llama.cpp  go-llama.cpp

RUN set -ex          ;\
    go version       ;\
    go mod download

COPY conf    conf
COPY files   files
COPY lm      lm
COPY server  server
COPY state   state
COPY types   types
COPY main.go .

COPY --from=infergui  dist  server/dist

# Go build flags
# "-s -w" removes all debug symbols: https://pkg.go.dev/cmd/link
# GOAMD64=v3 --> https://github.com/golang/go/wiki/MinimumRequirements#amd64
RUN set -ex                                          ;\
    ls -lA . server/dist                             ;\
    export GOFLAGS="-trimpath -modcacherw"           ;\
    export GOLDFLAGS="-d -s -w -extldflags=-static"  ;\
    export GOAMD64=v3                                ;\
    go build -v  .                                   ;\
    ls -lA                                           ;\
    ./goinfer -help       # smoke test


# --------------------------------------------------------------------
FROM docker.io/golang:1.21 AS integrator

WORKDIR /target

# Copy HTTPS root certificates (200 KB) + Create user/group files 
RUN set -ex                                                 ;\
    mkdir -p                                 etc/ssl/certs  ;\
    cp -a /etc/ssl/certs/ca-certificates.crt etc/ssl/certs  ;\
    echo "go:x:$uid:$uid::/:" > etc/passwd                  ;\
    echo "go:x:$uid:"         > etc/group

# Copy static website and backend executable
COPY --from=goinfer  /code/goinfer .

# Copies the dynamic libs
RUN set -ex                                           ;\
    ldd goinfer                                       ;\
    ldd goinfer                                       |\
    while read lib rest                               ;\
    do                                                 \
       find / -name "$lib"                            ;\
       find / -name "$lib" | while read path          ;\
       do mkdir -p /target"${path%/*}"      &&         \
          cp -v "$path" /target"${path%/*}" || true   ;\
       done                                           ;\
    done                                              ;\
    mv usr/lib lib                                    ;\
    rmdir usr                                         ;\
    mkdir -p lib64                                    ;\
    cp /usr/lib64/ld-linux-x86-64.so.2 lib64          ;\
    echo '{\n'                                                                              \
    '   "api_key": "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465",\n'   \
    '   "models_dir": "/home/me/my/lm/models",\n'                                           \
    '   "tasks_dir": "./tasks",\n'                                                          \
    '   "origins": [\n'                                                                     \
    '       "http://localhost:5173",\n'                                                     \
    '       "http://localhost:5143"\n'                                                      \
    '   ]\n'                                                                                \
    '}' > goinfer.config.json                                                              ;\
    ls -lA /target

# --------------------------------------------------------------------
FROM scratch AS final

COPY --chown=$uid:$uid --from=integrator /target /

# Run as unprivileged
USER $uid:$uid

# Use UTC time zone by default
ENV TZ UTC0

ENTRYPOINT ["/goinfer", "-local"]
