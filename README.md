# Goinfer

**Inference proxy** – swap between multiple `*.gguf` models on remote machines and expose them through HTTPS-API with credentials. So you can a securely connect from any device to your home GPU computers, or to let employees to connect to idle GPUs within the company office.

**TL;DR** – Deploy a **client** on a GPU-rich desktop, a **server** on a machine with a static IP (or DNS), and let the server forward inference requests to the client. No VPN, no port-forwarding, end-to-end encryption.

Built on top of [llama.cpp](https://github.com/ggml-org/llama.cpp) and [llama-swap](https://github.com/mostlygeek/llama-swap), Goinfer is designed to be DevOps-friendly, easily deployable/monitored on remote computers with the minimum manual operations (inspired by [llamactl](https://github.com/lordmathis/llamactl)), and meaningful logs.

## Problem: remote access to office/home-hosted LLM

⚠️ **Not yet implemented. Please contribute. Contact us team@LM4.eu** ⚠️

Local-LLM enthusiasts often hit a wall when they try to expose a model to the Internet:

- **Security** – exposing a raw `llama-server` or `ollama` instance can [leak the GPU to anyone](https://reddit.com/r/LocalLLaMA/comments/1nlpx3p).
- **Network topology** – most home routers block inbound connections, so the GPU machine can’t be *reached* from outside and home IP changes.
- **Privacy** – using third-party inference services defeats the purpose of running models locally.

Existing tools ([llamactl](https://github.com/lordmathis/llamactl), [llama-swap](https://github.com/mostlygeek/llama-swap), [olla](https://github.com/thushan/olla), [llm-proxy/rust](https://github.com/x5iu/llm-proxy), [llm-proxy/py](https://github.com/llm-proxy/llm-proxy), [langroute](https://github.com/bluewave-labs/langroute), [optillm](https://github.com/codelion/optillm), VPNs, WireGuard, SSH...) either require inbound ports, complex network plumbing, or a custom client on every device.

**Goinfer** solves these issues by flipping the connection direction: the GPU-rich **client** (home) *initiates* a secure outbound connection to a **server** with a static IP. The server then acts as a public façade, forwarding inference requests back to the client (home-hosted LLM).

## Key features

Category            | Feature
--------------------|----------
**Model handling**  | Load multiple `*.gguf` models, switch at runtime, change any inference parameter
**API**             | OpenAI-compatible HTTP API `/v1/`, LLama.cpp-compatible `/completions` API, streaming responses
**Security**        | API key, CORS control
**Robustness**      | Independent of ISP-provided IP, graceful reconnects
**Admin control**   | Remote monitoring, delete/upload new GGUF files, reload config, `git pull llama.cpp`, re-compile
**Home-hosted LLM** | Run Goinfer on your GPU desktop and another Goinfer in a data-center (static IP/DNS)

## Build

- [Go](https://gist.github.com/MichaelCurrin/ca6b3b955172ff993184d39807dd68d4) (any version, `go` will automatically use Go-1.25 to build Goinfer)
- GCC/LLVM if you want to build [llama.cpp](https://github.com/ggml-org/llama.cpp) or [ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp/) or …
- NodeJS (optional, llama.cpp frontend is already built)
- One or more `*.gguf` model files

### Container

See the [Containerfile](./Containerfile)
to build a Docker/Podman image
with official Nvidia images,
CUDA-13, GCC-14 and optimized CPU flags.

### First run

```bash
git clone https://github.com/LM4eu/goinfer
cd goinfer

# discover the parent directories of your GUFF files
#   - find the files *.gguf 
#   - -printf their folders (%h)
#   - sort them, -u to keep a unique copy of each folder
#   - while read xxx; do xxx; done  =>  print the parent folders separated by ":"
export GI_MODELS_DIR="$(find ~ /mnt -name '*.gguf' -printf '%h\0' | sort -zu |
while IFS= read -rd '' d; do [[ $p && $d == "$p"/* ]] && continue; echo -n "$d:"; p=$d; done)"

# set the path of your inference engine (llama.cpp/ik_llama.cpp/...)
export GI_LLAMA_EXE=/home/me/bin/llama-server

# generates the config
go run . -write

# voilà, it's running
go run . -no-api-key
```

Goinfer listens on the port defined in `goinfer.ini`.
Default port is `8080` using OpenAI-compatible API.

```sh
# use the default model
curl -X POST localhost:8080/completions -d '{"prompt":"Hello"}'

# list the models
curl -X GET localhost:8080/models | jq

# pick up a model and prompt
curl -X POST localhost:8080/completion \
  -d '{ "model":"qwen-3b", "prompt":"Hello AI" }'

# OpenAI API fashion
curl -X POST localhost:8080/v1/chat/completions \
  -d '{ "model": "qwen-3b",                     \
        "messages": [ {"role":"user",           \
                       "content":"Hello AI"}]   \
      }'
```

### All-in-one script

Build all dependencies and run Goinfer with the bash script
[`clone-pull-build-run.sh`](./scripts/clone-pull-build-run.sh)

    ```sh
    git clone https://github.com/LM4eu/goinfer
    goinfer/scripts/clone-pull-build-run.sh --build--swap
    ```

This script clones and builds [llama.cpp](https://github.com/ggml-org/llama.cpp)
using the best optimizations flags for your CPU.
This script also discovers your GGUF files:
your personalized configuration files is automatically generated
(no need to edit manually the configuration files).

Perfect to setup the environment,
and to daily update/build the [llama.cpp](https://github.com/ggml-org/llama.cpp) dependency.

The script ends by running a fully configured Goinfer server.

To reuse your own `llama-server` set:  
`export GI_LLAMA_EXE=/home/me/path/llama-server`
(this will prevent cloning/building the llama.cpp)

If this script finds too much `*.gguf` files, set:  
`export GI_MODELS_DIR=/home/me/models:/home/me/other/path`
(this will disable the GUFF search and speedup the script)

Run Goinfer in local without the API key:  
`./clone-pull-build-run.sh -no-api-key`

Full example:  

```sh
git -C path/repo/goinfer pull --ff-only
export GI_MODELS_DIR=/home/me/models
export GI_DEFAULT_MODEL=my-favorite-model.gguf
export GI_LLAMA_EXE=/home/me/bin/llama-server
path/repo/goinfer/scripts/clone-pull-build-run.sh -no-api-key
```

Use the flag `--help` or the usage within the [script](./scripts/clone-pull-build-run.sh).

## Configuration

### Environment variables

Discover the parent folders of your GUFF models:

- `find` the files `*.gguf` in `$HOME` and `/mnt`
- `-printf` their folders `%h` separated by nul character `\0` (support folder names containing newline characters)
- `sort` them, `-u` to keep a *unique* copy of each folder (`z` = input is `\0` separated)
- `while read xxx; do xxx; done` to keep the parent folders
- `echo $d:` prints each parent folder separated by `:` (`-n` no newline)

```bash
export GI_MODELS_DIR="$(find "$HOME" /mnt -type f -name '*.gguf' -printf '%h\0' | sort -zu |
while IFS= read -rd '' d; do [[ $p && $d == "$p"/* ]] && continue; echo -n "$d:"; p=$d; done)"

# else manually

export GI_MODELS_DIR=/path/to/my/models

# multiple paths

export GI_MODELS_DIR=/path1:/path2:/path3
```

`GI_MODELS_DIR` is the root path where your models are stored.
`goinfer` will search `*.gguf` files within all `GI_MODELS_DIR` sub-folders.
So you can organize your models within a folders tree.

The other environment variables are:

```sh
export GI_LLAMA_EXE=/path/to/my/llama-server
export GI_HOST=0.0.0.0  # expose Goinfer on your LAN
export GI_ORIGINS=      # disable CORS whitelist
export GI_API_KEY="PLEASE SET SECURE API KEY"
```

Disable Gin debug logs:

```sh
export GIN_MODE=release 
```

### API key

The flag `-write` also generates a random API key in `goinfer.ini`.
This flag can be combined with:

- `-debug` sets the debug API key (only during the dev cycle)

- `-no-api-key` sets the API key with "Please ⚠️ Set your API key"
    admin: "PLEASE

Set the Authorization header within the HTTP request:

```sh
curl -X POST https://localhost:8080/completions  \
  -H "Authorization: Bearer $GI_API_KEY"         \
  -d '{ "prompt": "Say hello in French" }'
```

### `goinfer.ini`

```ini
# ⚠️ Set your API key, can be 64-hex-digit (32-byte) 🚨
# Goinfer sets a random API key with: ./goinfer -write
api_key = '166a7c4bb8e9da0e1c414049c20797ec0fb9053d6bb553bf3f2dfcf1183451f5'
# 
# CORS whitelist (env. var: GI_ORIGINS)
origins = 'localhost'
# 
# Goinfer recursively searches GGUF files in one or multiple folders separated by ':'
# List your GGUF dirs with: locate .gguf | sed -e 's,/[^/]*$,,' | uniq
# env. var: GI_MODELS_DIR
models_dir = '/home/me/path/to/models'
# 
# The default model name to load at startup
# Can also be set with: ./goinfer -start <model-name>
default_model = ''

# Download models using llama-server flags
# see : github.com/ggml-org/llama.cpp/blob/master/common/arg.cpp#L3000
[extra_models]
'OuteAI/OuteTTS-0.2-500M-GGUF+ggml-org/WavTokenizer' = '--tts-oute-default'
'ggml-org/Qwen2.5-Coder-1.5B-Q8_0-GGUF' = '--fim-qwen-1.5b-default'
'ggml-org/Qwen2.5-Coder-14B-Q8_0-GGUF+0.5B-draft' = '--fim-qwen-14b-spec'
'ggml-org/Qwen2.5-Coder-3B-Q8_0-GGUF' = '--fim-qwen-3b-default'
'ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF' = '--fim-qwen-7b-default'
'ggml-org/Qwen2.5-Coder-7B-Q8_0-GGUF+0.5B-draft' = '--fim-qwen-7b-spec'
'ggml-org/Qwen3-Coder-30B-A3B-Instruct-Q8_0-GGUF' = '--fim-qwen-30b-default'
'ggml-org/embeddinggemma-300M-qat-q4_0-GGUF' = '--embd-gemma-default'
'ggml-org/gemma-3-12b-it-qat-GGUF' = '--vision-gemma-12b-default'
'ggml-org/gemma-3-4b-it-qat-GGUF' = '--vision-gemma-4b-default'
'ggml-org/gpt-oss-120b-GGUF' = '--gpt-oss-120b-default'
'ggml-org/gpt-oss-20b-GGUF' = '--gpt-oss-20b-default'

[llama]
# path of llama-server
exe = '/home/me/llama.cpp/build/bin/llama-server'
# common args used for every model
common = '--props --no-warmup --no-mmap'
# extra args to let tools like Agent-Smith doing the templating (/completions endpoint)
goinfer = '--jinja --chat-template-file template.jinja'
# extra llama-server flag when ./goinfer is used without the -q flag
verbose = '--verbose-prompt'
# extra llama-server flag for ./goinfer -debug
debug = '--verbosity 3'
# address can be 'host:port' or 'ip:por' or simply ':port' (for host = localhost)
addr = ':8080' # OpenAI-compatible API
```

- **API key** – Never commit them. Use env. var. `GI_API_KEY` or a secrets manager in production.
- **Origins** – Set to the domains you’ll be calling the server from (including `localhost` for testing).
- **Ports** – Adjust as needed; make sure the firewall on the server allows them.

### `llama-swap.yml`

At startup, Goinfer verifies the available GUFF files.
The flag `-write` tells Goinfer to write the `llama-swap.yml` file.

Official documentation:
[github/mostlygeek/llama-swap/wiki/Configuration](https://github.com/mostlygeek/llama-swap/wiki/Configuration)

```yaml
logLevel: info            # debug, info, warn, error
healthCheckTimeout: 500   # seconds to wait for a model to become ready
metricsMaxInMemory: 1000  # maximum number of metrics to keep in memory
startPort: 6000           # first ${PORT} incremented for each model

macros:  # macros to reduce common conf settings
    cmd-fim: /home/me/llama.cpp/build/bin/llama-server --props --no-warmup --no-mmap --verbose-prompt
    cmd-common: ${cmd-fim} --jinja --port ${PORT}
    cmd-goinfer: ${cmd-common} --chat-template-file template.jinja

models:

  # model name used in API requests
  ggml-org/Qwen2.5-Coder-0.5B-Q8_0-GGUF_qwen2.5-coder-0.5b-q8_0:
    description: "Small but capable model for quick testing"
    name: Qwen2.5-Coder-0.5B-Q8_0-GGUF  # for /v1/models response
    useModelName: "Qwen2.5-Coder"       # overrides the model name for /upstream (used by llama-swap web UI)
    aliases:
      - "Qwen2.5-Coder-0.5B-Q8_0"       # alternative names (unique globally)
      - "Qwen2.5-Coder-0.5B"
    env: []
    cmd: ${cmd-common}  -m /home/c/.cache/llama.cpp/ggml-org_Qwen2.5-Coder-0.5B-Q8_0-GGUF_qwen2.5-coder-0.5b-q8_0.gguf
    proxy: http://localhost:${PORT}     # default: http://localhost:${PORT}
    checkEndpoint: /health              # default: /health endpoint
    unlisted: false                     # unlisted=false => list model in /v1/models and /upstream responses
    ttl: 3600                           # stop the cmd after 1 hour of inactivity
    filters:
      # inference params to remove from the request, default: ""
      # useful for preventing overriding of default server params by requests
      strip_params: "temperature,top_p,top_k"

  # A_ prefix for Agent-Smith compatibility
  A_ggml-org/Qwen2.5-Coder-0.5B-Q8_0-GGUF_qwen2.5-coder-0.5b-q8_0:
      cmd: ${cmd-goinfer}  -m /home/c/.cache/llama.cpp/ggml-org_Qwen2.5-Coder-0.5B-Q8_0-GGUF_qwen2.5-coder-0.5b-q8_0.gguf
      proxy: http://localhost:${PORT}
      checkEndpoint: /health
      unlisted: true   # hide model name in /v1/models and /upstream responses
      useModelName: ggml-org/Qwen2.5-Coder-0.5B-Q8_0-GGUF # for /upstream (used by llama-swap web UI)

  # selected models by llama.cpp are also available with their specific port
  ggml-org/Qwen2.5-Coder-1.5B-Q8_0-GGUF:
      cmd: ${cmd-fim} --fim-qwen-1.5b-default
      proxy: http://localhost:8012
      checkEndpoint: /health
      unlisted: false

# preload some models on startup 
hooks:
  on_startup:
    preload:
      - "Qwen2.5-1.5B-Instruct-Q4_K_M"

# Keep some models loaded indefinitely, while others are swapped out
# see https://github.com/mostlygeek/llama-swap/pull/109
groups:
  # example1: only one model is allowed to run a time (default mode)
  "group1":
    swap: true
    exclusive: true
    members:
      - "llama"
      - "qwen-unlisted"
  # example2: all the models in this group2 can run at the same time
  # loading another model => unloads all this group2
  "group2":
    swap: false
    exclusive: false
    members:
      - "docker-llama"
      - "modelA"
      - "modelB"
  # example3: persistent models are never unloaded
  "forever":
    persistent: true
    swap: false
    exclusive: false
    members:
      - "forever-modelA"
      - "forever-modelB"
      - "forever-modelC"
```

## Developer info

- flags override environment variables that override YAML config: `Cfg` defined in [`conf.go`](go/conf/conf.go)
- GUFF files discovery: `Search()` in [`models.go`](go/conf/models.go)
- Graceful shutdown handling: `handleShutdown()` in [`goinfer.go`](go/goinfer.go)
- API-key authentication per service: `configureAPIKeyAuth()` in [`router.go`](go/infer/router.go)
- Comprehensive error handling: `gie` package in [`errors.go`](go/gie/errors.go)

## API endpoints

Method | Path                   | Description
-------|------------------------|------------
GET    | `/`                    | llama.cpp Web UI
GET    | `/ui`                  | llama-swap Web UI
GET    | `/models`              | List available GGUF models
POST   | `/completions`         | Llama.cpp inference API
GET    | `/v1/models`           | List models by llama-swap
POST   | `/v1/chat/completions` | OpenAI-compatible chat endpoint
POST   | `/v1/*`                | Other OpenAI endpoints
POST   | `/rerank` `/v1/rerank` | Reorder or answer questions about a document
POST   | `/infill`              | Auto-complete source code (or other edition)
GET    | `/logs` `/logs/stream` | Retrieve the llama-swap or llama.cpp logs
GET    | `/props`               | Get the llama.cpp settings
GET    | `/unload`              | Stop all inference engines
GET    | `/running`             | List the running inference engines
GET    | `/health`              | Check if everything is OK

Goinfer endpoints require an `Authorization: Bearer $GI_API_KEY` header (disabled by `-no-api-key` flag).

llama-swap starts `llama-server` using the command lines configured in `llama-swap.yml`.
Goinfer generates that `llama-swap.yml` file setting two different command lines for each model:

1. classic command line for models listed by `/v1/models` (to be used by tools like Cline / RooCode)
2. with extra arguments `--jinja --chat-template-file template.jinja` when the requested model is prefixed with `A_`

The first one is suitable for most of the use cases such as Cline / RooCode.
The second one is a specific use case for tools like
[`agent-smith`](https://github.com/synw/agent-smith)
requiring full inference control (e.g. no default Jinja template).

## Server/Client mode

⚠️ **Not yet implemented. Please contribute. Contact us team@LM4.eu** ⚠️

### Design

    ╭──────────────────┐  1 ──>  ╭───────────────────┐         ╭──────────────┐
    │ GPU-rich desktop │         │ host static IP/DNS│  <── 2  │ end-user app │
    │ (Goinfer client) │  <── 3  │  (Goinfer server) │         │ (browser/API)│
    └──────────────────╯  4 ──>  └───────────────────╯  5 ──>  └──────────────╯

1. Goinfer client connects to the Goinfer server having a static IP (and DNS)
2. the end user sends a prompt to the cloud-hosted Goinfer server
3. the Goinfer server reuses the connection to the Goinfer client and forwards it the prompt
4. the Goinfer client reply the processed prompt by the local LLM using llama.cpp
5. the Goinfer server forwards the response to the end-user

No inbound ports are opened on neither the Goinfer client nor the end-user app,
maximizing security and anonymity between the GPU-rich desktop and the end-user.

Another layer of security is the encrypted double authentication
between the Goinfer client and the Goinfer server.
Furthermore, we recommend to use HTTPS on port 443
for all these communications to avoid sub-domains
because sub-domains remain visible over HTTPS, not URL paths.

High availability is provided by the multiple-clients/multiple-servers architecture:

- The end-user app connects to one of the available Goinfer servers.
- All the running Goinfer clients connects to all the Goinfer servers.
- The Goinfer servers favor the most idle Goinfer clients depending on their capacity
  (vision prompts are sent to GPU-capable clients running the adequate LLM).
- Fallback to CPU offloading when appropriate.

### 1. Run the **server** (static IP / DNS)

On a VPS, cloud VM, or any machine with a public address.

```bash
./goinfer
```

### 2. Run the **client** (GPU machine)

On your desktop with a GPU

```bash
./goinfer
```

The client will connect, register its available models and start listening for inference requests.

### 3. Test the API

```sh
# list the available models
curl -X GET https://my-goinfer-server.com/v1/models

# pick up a model and prompt it
curl -X POST https://my-goinfer-server.com/v1/chat/completions \
  -H "Authorization: Bearer $GI_API_KEY"                       \
  -d '{                                                        \
        "model":"Qwen2.5-1.5B-Instruct-Q4_K_M",                \
        "messages":[{"role":"user",                            \
                     "content":"Say hello in French"}]         \
      }'
```

You receive a JSON response generated by the model running on your GPU rig.

## History & Roadmap

### March 2023

In March 2023, [Goinfer](https://github.com/LM4eu/goinfer)
was an early local LLM proxy swapping models and supporting Ollama, Llama.cpp, and KoboldCpp.
Goinfer has been initiated for two needs:

1. to swap engine and model at runtime, something that didn’t exist back then
2. to infer pre-configured templated prompts

This second point has been moved to the project
[github/synw/agent-smith](https://github.com/synw/agent-smith)
with more templated prompts in
[github/synw/agent-smith-plugins](https://github.com/synw/agent-smith-plugins).

### August 2025

To simplify the maintenance, we decided in August 2025
to replace our process management with another well-maintained project.
As we do not use Ollama/KoboldCpp any more,
we integrated [llama-swap](https://github.com/mostlygeek/llama-swap)
into Goinfer to handle communication with `llama-server`.

### New needs

Today the needs have evolved. We need most right now is a proxy that can act as a secure intermediary between a **client (frontend/CLI)** and **a inference engine (local/cloud)** with these these constrains:

Client   | Server       | Constraint
---------|--------------|-----------
Frontend | OpenRouter   | Intermediate proxy required to manage the OpenRouter key without exposing it on the frontend
Any      | Home GPU rig | Access to another home GPU rig that forbids external TCP connections

### Next implementation

Integrate a Web UI to select the model(s) to enable.

Optimizer of the [`llama-server` command line](https://github.com/ggml-org/llama.cpp/tree/master/tools/server#usage): finding the best `--gpu-layers --override-tensor --n-cpu-moe --ctx-size ...` by iterating of GPU allocation error and benching. Timeline of llama.cpp optimization:
- Apr 2024 llama.cpp [PR `-ngl auto`](https://github.com/ggml-org/llama.cpp/pull/6502) (Draft)
- Jan 2025 [study](https://github.com/robbiemu/llama-gguf-optimize
- Mar 2025 [Python script determining `-ngl`](https://github.com/fredlas/optimize_llamacpp_ngl)
- Jun 2025 llama.cpp [another PR](https://github.com/ggml-org/llama.cpp/pull/14067) (Draft) based on these [ideas](https://github.com/ggml-org/llama.cpp/issues/13860)
- Jun 2025 [Python script running `llama-bench` for best `-b -ub -fa -t -ngl -ot`](https://github.com/BrunoArsioli/llama-optimus) ([maybe integrated in llamap.cpp](https://github.com/ggml-org/llama.cpp/discussions/14191)) 

This last Python script `llama-optimus` is nice and could also be used for `ik_llama.cpp`. Its [README](https://github.com/BrunoArsioli/llama-optimus?tab=readme-ov-file#coments-about-other-llamacpp-flags) explains:

Flag | Why it matters | Suggested search values | Notes
---- |---- |---- |----
**`--mmap / --no-mmap`** (memory-map model vs. fully load) | • On fast NVMe & Apple SSD, `--mmap 1` (default) is fine.<br>• On slower HDD/remote disks, disabling mmap (`--no-mmap` or `--mmap 0`) and loading the whole model into RAM often gives **10-20 % faster generation** (no page-fault stalls).   | `[0, 1]` (boolean)  | Keep default `1`; let Optuna see if `0` wins on a given box.
**`--cache-type-k / --cache-type-v`** | Setting key/value cache to **`f16` vs `q4`** or **`i8`** trades RAM vs speed.  Most Apple-Metal & CUDA users stick with `f16` (fast, larger).  For low-RAM CPUs increasing speed is impossible if it swaps; `q4` can shrink cache 2-3× at \~3-5 % speed cost. | `["f16","q4"]` for both k & v (skip i8 unless you target very tiny devices). | Only worth searching when the user is on **CPU-only** or small-VRAM GPU. You can gate this by detecting “CUDA not found” or VRAM < 8 GB.
**`--main-gpu`** / **`--gpu-split`** (or `--tensor-split`) | Relevant only for multi-GPU rigs (NVIDIA).  Picking the right primary or a tensor split can cut VRAM fragmentation and enable higher `-ngl`. | If multi-GPU detected, expose `[0,1]` for `main-gpu` **and** a handful of tensor-split presets (`0,1`, `0,0.5,0.5`, etc.). | Keep disabled on single-GPU/Apple Silicon to avoid wasted trials.
**[Preliminary] `--flash-attn-type 0/1/2`** (v0.2+ of llama.cpp)         | Metal + CUDA now have two flash-attention kernels (`0` ≈ old GEMM, `1` = FMHA, `2` = in-place FMHA). **!!Note!!:** Not yet merged to llama.cpp main.  Some M-series Macs get +5-8 % with type 2 vs 1. | `[0,1,2]` —but **only if llama.cpp commit ≥ May 2025**.  | Add a version guard: skip the flag on older builds.

When the VRAM is not enouth or when the user needs to increase the context size,
Goinfer needs to offload some layers to CPU.
The idea is to identify the least used tensors and to offload them in priority.
The command `llama-gguf` lists the tensors (experts are usually suffixed with `_exps`).

### Other priority task

Two Goinfer instances (client / server mode):

- a Goinfer on a GPU machine that runs in client mode  
- a Goinfer on a machine in a data-center (static IP) that runs in server mode  
- the client Goinfer connects to the server Goinfer (here, the server is the backend of a web app)  
- the user sends their inference request to the backend (data-center) which forwards it to the client Goinfer  
- we could imagine installing a client Goinfer on every computer with a good GPU, and the server Goinfer that forwards inference requests to the connected client Goinfer according to the requested model

### Medium priority

Manage the OpenRouter API key of a AI-powered frontend.

### Low priority

- Comprehensive **web admin** (monitoring, download/delete `.gguf`, edit config, restart, `git pull` + rebuild `llama.cpp`, remote shell, upgrade Linux, reboot the machine, and other SysAdmin tasks)

> **contribute** – If you’re interested in any of the above, open an issue or submit a PR :)

### Prompting creator UI

Integrate a Web UI to ease creation of multi-step AI agents like:

- [github/synw/agent-smith](https://github.com/synw/agent-smith)
- [n8n](https://n8n.io/)
- [flowiseai](https://flowiseai.com/)
- tools from [github/HKUDS](https://github.com/HKUDS)

The angentic prompt syntax [YALP](https://yapl-language.github.io) and its [VS-Code extension](https://github.com/yapl-language/yapl-vscode) authored by two Germans: [Nils Abegg](https://github.com/nilsabegg) et [einfachai](https://github.com/einfachai) (see also [einfachai.com](https://einfachai.com)).

### Nice to have

Some inspiration to extend the Goinfer stack:

- [`compose.yml`](./compose.yml) with something like [github/j4ys0n/local-ai-stack](https://github.com/j4ys0n/local-ai-stack) and [github/LLemonStack/llemonstack](https://github.com/LLemonStack/llemonstack)
- WebUI [github/oobabooga/text-generation-webui](https://github.com/oobabooga/text-generation-webui), [github/danny-avila/LibreChat](https://github.com/danny-avila/LibreChat), [github/JShollaj/Awesome-LLM-Web-UI](https://github.com/JShollaj/Awesome-LLM-Web-UI)
- Vector Database and Vector Search Engine [github/qdrant/qdrant](https://github.com/qdrant/qdrant)
- Convert an webpage (URL) into clean markdown or structured data [github/firecrawl/firecrawl](https://github.com/firecrawl/firecrawl) [github/unclecode/crawl4ai](https://github.com/unclecode/crawl4ai) [github/browser-use/browser-use](https://github.com/browser-use/browser-use)
- [github/BerriAI/litellm](https://github.com/BerriAI/litellm) + [github/langfuse/langfuse](https://github.com/langfuse/langfuse)
- [github/claraverse-space/ClaraCore](https://github.com/claraverse-space/ClaraCore) automatizes installation & configuration of llama-swap

## Contributions welcomed

1. Fork the repository.
2. Create a feature branch (`git checkout -b your-feature`)
3. Run the test suite: `go test ./...` (more tests are welcome)
4. Ensure code is formatted and linted with `golangci-lint-v2 run --fix`
5. Submit a PR with a clear description and reference any related issue

Feel free to open discussions for design ideas/decisions.

## License

- **License:** MIT – see [`LICENSE`](./LICENSE) file.
- **Dependencies:**
  - [llama.cpp](https://github.com/ggml-org/llama.cpp) – Apache-2.0
  - [llama-swap](https://github.com/LM4eu/llama-swap) – MIT

## Merci

Special thanks to:

- [Georgi Gerganov](https://github.com/ggerganov) for releasing and improving [llama.cpp](https://en.wikipedia.org/wiki/Llama.cpp) in 2023 so we could freely play with Local LLM.
- All other contributors of [llama.cpp](https://en.wikipedia.org/wiki/Llama.cpp).
- [Benson Wong](https://github.com/mostlygeek) for maintaining [llama-swap](https://github.com/mostlygeek/llama-swap) with clean and well-documented code.
- The open-source community that makes GPU-based LLM inference possible on commodity hardware. :heart:

## See also

Some active local-LLM proxies:

Language   | Repository
-----------|---------------
Go         | [github/inference-gateway/inference-gateway](https://github.com/inference-gateway/inference-gateway)
Go         | [github/lordmathis/llamactl](https://github.com/lordmathis/llamactl)
Go         | [github/mostlygeek/llama-swap](https://github.com/mostlygeek/llama-swap)
Go         | [github/thushan/olla](https://github.com/thushan/olla)
Python     | [github/codelion/optillm](https://github.com/codelion/optillm)
Python     | [github/llm-proxy/llm-proxy](https://github.com/llm-proxy/llm-proxy) (inactive?)
Rust       | [github/x5iu/llm-proxy](https://github.com/x5iu/llm-proxy)
TypeScript | [github/bluewave-labs/langroute](https://github.com/bluewave-labs/langroute)

Compared to alternatives, we like [llama-swap](https://github.com/mostlygeek/llama-swap) for its readable source code and because its author contributes regularly. So we integrated it into Goinfer to handle communication with `llama-server` (or other compatible forks as [ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp/)). We also like [llamactl](https://github.com/lordmathis/llamactl) ;-)

**Enjoy remote GPU inference with Goinfer!** 🚀

*If you have questions, need help setting up your first client/server pair, or want to discuss future features, open an issue or ping us on the repo’s discussion board.*
