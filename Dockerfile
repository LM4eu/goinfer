# Build:
#
#    docker  build -t goinfer .
#    podman  build -t goinfer .
#    buildah build -t goinfer .
#
# Run:
#
#    docker run --rm -p 5143:5143 -v $PWD/goinfer.config.json:/goinfer.config.json goinfer
#    podman run --rm -p 5143:5143 -v $PWD/goinfer.config.json:/goinfer.config.json goinfer


# Arguments:
#
# pass the argument at build time:
#
#    docker build -t goinfer --build-arg uid=1122 .
#
# uid : to run goinfer as unprivileged (rootless)
ARG uid=5505


# --------------------------------------------------------------------
# https://hub.docker.com/_/node
FROM docker.io/node:21-bookworm AS infergui

WORKDIR /code

# Clone repo + build + clean
RUN set -ex                                        ;\
    git --version                                  ;\
    git clone https://github.com/synw/infergui .   ;\
    ls -lShA                                       ;\
    yarn versions                                  ;\
    yarn install --frozen-lockfile                 ;\
    yarn cache clean                               ;\
    ls -lShA                                       ;\
    yarn build                                     ;\
    mv /code/dist /dist                            ;\
    rm -r /code


# --------------------------------------------------------------------
# https://hub.docker.com/_/golang
FROM docker.io/golang:1.22 AS llama

WORKDIR /code

RUN git clone --recurse-submodules https://github.com/go-skynet/go-llama.cpp

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && apt install -y patch cmake && rm -rf /var/lib/apt/lists/*

RUN make -C go-llama.cpp libbinding.a -j $(nbcores)



# --------------------------------------------------------------------
FROM docker.io/golang:1.22 AS goinfer

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

# Go build flags: "-s -w" removes all debug symbols: https://pkg.go.dev/cmd/link
# GOAMD64=v3 --> https://github.com/golang/go/wiki/MinimumRequirements#amd64
RUN set -ex                                          ;\
    ls -lShA . server/dist                           ;\
    export CGO_ENABLED=0                             ;\
    export GOFLAGS="-trimpath -modcacherw"           ;\
    export GOLDFLAGS="-d -s -w -extldflags=-static"  ;\
    export GOAMD64=v3                                ;\
    export GOEXPERIMENT=newinliner                   ;\
    go build -a -v  .                                ;\
    ls -lShA                                         ;\
    ./goinfer -help       # smoke test


# --------------------------------------------------------------------
FROM docker.io/golang:1.22 AS integrator

WORKDIR /target

ARG uid

# Copy HTTPS root certificates (adds about 200 KB)
# and create user & group files
RUN set -ex                                                 ;\
    mkdir -p                                 etc/ssl/certs  ;\
    cp -a /etc/ssl/certs/ca-certificates.crt etc/ssl/certs  ;\
    echo "go:x:$uid:$uid::/:" > etc/passwd                  ;\
    echo "go:x:$uid:"         > etc/group

# Copy static website and backend executable
COPY --from=goinfer  /code/goinfer .

# Copy the dynamic libs
RUN set -ex                                           ;\
    ldd goinfer                                       ;\
    ldd goinfer |                                      \
    while read lib rest                               ;\
    do                                                 \
       find / -path /proc -prune -o -name "$lib" |     \
       while read path                                ;\
       do mkdir -p /target"${path%/*}"      &&         \
          cp -v "$path" /target"${path%/*}" || true   ;\
       done                                           ;\
    done                                              ;\
    mv usr/lib lib                                    ;\
    rmdir usr                                         ;\
    mkdir -p lib64                                    ;\
    cp /usr/lib64/ld-linux-x86-64.so.2 lib64          ;\
    ls -lShA /target

# --------------------------------------------------------------------
FROM scratch AS final

# Run as unprivileged
ARG    uid
USER "$uid:$uid"

# In this tiny image, put only the executable "goinfer",
# its lib dependencies, the SSL certificates,
# the "passwd" and "group" files. No shell commands.
COPY --chown=$uid:$uid --from=integrator /target /

# Default timezone is UTC
ENV TZ UTC0

# The default command to run the container
ENTRYPOINT ["/goinfer"]

# Default argument(s) appended to ENTRYPOINT
CMD ["-local"]