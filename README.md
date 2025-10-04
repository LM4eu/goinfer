# Goinfer

**Inference proxy for local LLMs** – run multiple `*.gguf` models on any machine, expose them through a secure HTTP‑API and optionally forward requests from a data‑center to idle GPUs in your home.

Built on top of **[llama.cpp](https://github.com/ggml-org/llama.cpp)** and **[llama‑swap](https://github.com/mostlygeek/llama-swap)**.

**TL;DR** – Deploy a **client** on a GPU‑rich desktop, a **server** on a machine with a static IP (or DNS), and let the server forward inference requests to the client. No VPN, no port‑forwarding, end‑to‑end encryption.

## problem: Remote access to home-hosted LLM

⚠️ **Not yet implemented** ⚠️

Local‑LLM enthusiasts often hit a wall when they try to expose a model to the Internet:

- **Security** – exposing a raw `llama-server` or `ollama` instance can [leak the GPU to anyone](https://reddit.com/r/LocalLLaMA/comments/1nlpx3p).
- **Network topology** – most home routers block inbound connections, so the GPU machine can’t be *reached* from outside and home IP changes.
- **Privacy** – using third‑party inference services defeats the purpose of running models locally.

Existing tools ([llamactl](https://github.com/lordmathis/llamactl), [llama‑swap](https://github.com/mostlygeek/llama-swap), [olla](https://github.com/thushan/olla), [llm-proxy/rust](https://github.com/x5iu/llm-proxy), [llm-proxy/py](https://github.com/llm-proxy/llm-proxy), [langroute](https://github.com/bluewave-labs/langroute), [optillm](https://github.com/codelion/optillm), VPNs, WireGuard, SSH...) either require inbound ports, complex network plumbing, or a custom client on every device.

**Goinfer** solves these issues by flipping the connection direction: the GPU‑rich **client** (home) *initiates* a secure outbound connection to a **server** with a static IP. The server then acts as a public façade, forwarding inference requests back to the client (home-hosted LLM).

## key features

Category            | Feature
--------------------|----------
**Model handling**  | Load multiple `*.gguf` models, switch at runtime, change any inference parameter
**API**             | OpenAI‑compatible HTTP API `/v1/`, LLama.cpp-compatible `/completions` API, streaming responses
**Security**        | API key, CORS control
**Robustness**      | Independent of ISP‑provided IP, graceful reconnects
**Admin control**   | Remote monitoring, delete/upload new GGUF files, reload config, `git pull llama.cpp`, re‑compile
**Home-hosted LLM** | Run Goinfer on your GPU desktop and another Goinfer in a data‑center (static IP/DNS)

## build

- [Go](https://gist.github.com/MichaelCurrin/ca6b3b955172ff993184d39807dd68d4) (any version, `go` will automatically use Go-1.25 to build Goinfer)
- GCC/LLVM if you want to build [llama.cpp](https://github.com/ggml-org/llama.cpp) or [ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp/) or …
- NodeJS (optional, llama.cpp frontend is already built)
- One or more `*.gguf` model files

### container

See the [Containerfile](./Containerfile)
to build a Docker/Podman image
with official Nvidia images,
CUDA-13, GCC-14 and optimized CPU flags.

### first run

```bash
git clone https://github.com/LM4eu/goinfer
cd goinfer/go

# discover the parent directories of your GUFF files
export GI_MODELS_DIR="$(find ~ /mnt -name '*.gguf' -printf '%h\0' | sort -zu |
while read -d '' d; do [[ $p && $d == $p/* ]] && continue; echo -n $d: ; p=$d; done)"

# set the path of your inference engine (llama.cpp/ik_llama.cpp/...)
export GI_LLAMA_EXE=/home/me/bin/llama-server

# generates the config (you may want to edit it)
go run . -gen

# voilà, it's running
go run . -no-api-key
```

Goinfer listens on the ports defined in `goinfer.yml`.
Default ports:

- `:4444` for extra-featured endpoints `/models`, `/completions`, `/v1/chat/completions`
- `:5555` for OpenAI‑compatible API (provided by llama-swap)

```sh
# use the default model
curl -X POST localhost:4444/completions -d '{"prompt":"Hello"}'

# list the models
curl -X GET localhost:4444/models | jq

# pick up a model and prompt it
curl -X POST localhost:4444/completion \
  -d '{ "model":"qwen-3b", "prompt":"Hello AI" }'

# same using the OpenAI API
curl -X POST localhost:5555/v1/chat/completions \
  -d '{ "model": "qwen-3b",                     \
        "messages": [ {"role":"user",           \
                       "content":"Hello AI"}]   \
      }'
```

### all-in-one script

Build all dependencies and run Goinfer with the bash script
[`clone-pull-build-run.sh`](./scripts/clone-pull-build-run.sh)

- clone and build [llama.cpp](https://github.com/ggml-org/llama.cpp) using optimizations flags
- clone and build the [llama‑swap](https://github.com/LM4eu/llama-swap) frontend with `--build--swap`:

    ```sh
    git clone https://github.com/LM4eu/goinfer
    goinfer/scripts/clone-pull-build-run.sh --build--swap
    ```

Perfect to setup the environment,
and to update/build daily the dependencies.

No need to edit manually the configuration files:
this script discovers your GGUF files.
Your personalized configuration files is automatically generated.

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
`GI_LLAMA_EXE=/home/me/bin/llama-server GI_MODELS_DIR=/home/me/models ./clone-pull-build-run.sh -no-api-key`

Use the flag `--help` or the usage within the [script](./scripts/clone-pull-build-run.sh).

## configuration

### environment variables

Discover the parent directories of your GUFF models:

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
export GI_HOST=0.0.0.0  # exposing llama-server is risky
export GI_ORIGINS=      # disabling CORS is risky
export GI_API_KEY="PLEASE SET SECURE API KEY"
```

Disable Gin debug logs:

```sh
export GIN_MODE=release 
```

### API key

The flag `-gen` also generates a random API key in `goinfer.yml`.
This flag can be combined with:

- `-debug` sets the debug API key (only during the dev cycle)

- `-no-api-key` sets the API key with "Please ⚠️ Set your API key"
    admin: "PLEASE

Set the Authorization header within the HTTP request:

```sh
curl -X POST https://localhost:4444/completions  \
  -H "Authorization: Bearer $GI_API_KEY"         \
  -d '{ "prompt": "Say hello in French" }'
```

### `goinfer.yml`

```yaml
# Goinfer recursively search GGUF files in one or multiple folders separated by ':'
# List your GGUF dirs with `locate .gguf | sed -e 's,/[^/]*$,,' | uniq`
models_dir: /home/me/models 

# ⚠️ Set your API key, can be 64‑hex‑digit (32‑byte) 🚨
# Generate these random API key with: ./goinfer -gen
api_key: "PLEASE SET USER API KEY"
origins:   # CORS whitelist
  - "https://my‑frontend.example.com"
  - "http://localhost"
listen:
  # format:  <address>: <list of enabled services>
  # <address> can be <ip|host>:<port> or simply :<port> when <host> is localhost
  ":4444": infer        # historical goinfer endpoints
  ":5555": llama-swap  # OpenAI‑compatible API by llama‑swap

llama:
  exe: /home/me/llama.cpp/build/bin/llama-server
  args:
    # common args used for every model
    common: "--props --no-warmup"
    # extra args for the completion endpoint (Jinja templating)
    infer: "--jinja --chat-template-file template.jinja"
```

- **API key** – Never commit them. Use env. var. `GI_API_KEY` or a secrets manager in production.
- **Origins** – Set to the domains you’ll be calling the server from (including `localhost` for testing).
- **Ports** – Adjust as needed; make sure the firewall on the server allows them.

### `llama‑swap.yml`

At startup, Goinfer verifies the available GUFF files
and generates the `llama‑swap.yml` file.

Official documentation:
[github.com/mostlygeek/llama-swap/wiki/Configuration](https://github.com/mostlygeek/llama-swap/wiki/Configuration)

```yaml
logLevel: info            # debug, info, warn, error
healthCheckTimeout: 500   # seconds to wait for a model to become ready
metricsMaxInMemory: 1000  # maximum number of metrics to keep in memory
startPort: 6000           # first ${PORT} incremented for each model

macros:  # macros to reduce common conf settings
  "cmd-openai": "./llama-server --port ${PORT} --props --no-webui --no-warmup"
  "cmd-infer": "./llama-server --port ${PORT} --props --no-webui --no-warmup --jinja --chat-template-file template.jinja"

models:
  "Qwen2.5-1.5B-Instruct-Q4_K_M":  # model names used in API requests
    aliases:                       # alternative model names (unique globally)
      - "Qwen2.5-1.5B-Instruct"    #     for impersonating a specific model
      - "Qwen2.5-1.5B"
    unlisted: false                # hide model name in /v1/models and /upstream API response
    name: "Qwen2.5 1.5B"           # name for human in /v1/models API response
    useModelName: "qwen:qwq"       # overrides the model name that is sent to /upstream server
    description: "Small but capable model for quick testing"
    env: []
    cmd: ${cmd-openai} --model /path/to/Qwen2.5-1.5B-Instruct-Q4_K_M.gguf
    proxy: http://localhost:${PORT}  # default: http://localhost:${PORT}
    checkEndpoint: /health           # default: /health
    ttl: 3600                        # stop the cmd after 1 hour of inactivity
    filters:
      # inference params to remove from the request, default: ""
      # useful for preventing overriding of default server params by requests
      strip_params: "temperature,top_p,top_k"

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

## developer info

- flags override environment variables that override YAML config: `Cfg` defined in [`conf.go`](go/conf/conf.go)
- GUFF files discovery: `Search()` in [`models.go`](go/conf/models.go)
- Graceful shutdown handling: `handleShutdown()` in [`goinfer.go`](go/goinfer.go)
- API‑key authentication per service: `configureAPIKeyAuth()` in [`router.go`](go/infer/router.go)
- Comprehensive error handling: `gie` package in [`errors.go`](go/gie/errors.go)

## API endpoints

Each service can be enabled/disabled in `goinfer.yml`.

Path                  | Method | Description
----------------------|--------|------------
`/`                   |  GET   | llama.cpp Web UI
`/ui`                 |  GET   | llama-swap Web UI
`/models`             |  GET   | List available GGUF models
`/completions`        |  POST  | Llama.cpp inference API
`/v1/models`          |  GET   | List models from Swap config
`/v1/chat/completions`|  POST  | OpenAI‑compatible chat endpoint
`/v1/*`               |  POST  | Other OpenAI endpoints

All endpoints require an `Authorization: Bearer $GI_API_KEY` header.

llama-swap starts `llama-server` using the command lines configured in `llama-swap.yml`.
Goinfer generates that `llama-swap.yml` file setting two different command lines for each model:

1. classic command line for models listed by `/v1/models` (to be used by tools like Cline / RooCode)
2. with extra arguments `--jinja --chat-template-file template.jinja` when the requested model is prefixed with `GI_`

The first one is suitable for most of the use cases such as Cline / RooCode.
The second one is a specific use case for tools like
[`agent-smith`](https://github.com/synw/agent-smith)
requiring full inference control (e.g. no default Jinja template).

## server/client mode

⚠️ **Not yet implemented** ⚠️

### design

    ╭──────────────────┐  1 ──>  ╭───────────────────┐         ╭──────────────┐
    │ GPU‑rich desktop │         │ host static IP/DNS│  <── 2  │ end‑user app │
    │ (Goinfer client) │  <── 3  │  (Goinfer server) │         │ (browser/API)│
    └──────────────────╯  4 ──>  └───────────────────╯  5 ──>  └──────────────╯

1. Goinfer client connects to the Goinfer server having a static IP (and DNS)
2. the end user sends a prompt to the cloud-hosted Goinfer server
3. the Goinfer server reuses the connection to the Goinfer client and forwards it the prompt
4. the Goinfer client reply the processed prompt by the local LLM using llama.cpp
5. the Goinfer server forwards the response to the end‑user

No inbound ports are opened on neither the Goinfer client nor the end-user app,
maximizing security and anonymity between the GPU‑rich desktop and the end‑user.

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
was an early local LLM proxy swapping models and supporting
Ollama, Llama.cpp, and KoboldCpp.
Goinfer has been initiated for two needs:

1. to swap engine and model at runtime, something that didn’t exist back then
2. to infer pre-configured templated prompts

This second point has been moved to the project
[github.com/synw/agent-smith](https://github.com/synw/agent-smith)
with more templated prompts in
[github.com/synw/agent-smith-plugins](https://github.com/synw/agent-smith-plugins).

### August 2025

To simplify the maintenance, we decided in August 2025
to replace our process management with another well-maintained project.
As we do not use Ollama/KoboldCpp any more,
we integrated [llama-swap](https://github.com/mostlygeek/llama-swap)
into Goinfer to handle communication with `llama-server`.

### October 2025

Restored `/completions` endpoint for full inference parameters control.

### New needs

Today the needs have evolved. We need most right now is a proxy that can act as a secure intermediary between a **client (frontend/CLI)** and **a inference engine (local/cloud)** with these these constrains:

Client   | Server       | Constraint
---------|--------------|-----------
Frontend | OpenRouter   | Intermediate proxy required to manage the OpenRouter key without exposing it on the frontend
Any      | Home GPU rig | Access to another home GPU rig that forbids external TCP connections

### High Priority (✅ in progress)

Manage the OpenRouter API key of a AI-powered frontend.

### Medium Priority

Two `goinfer` instances (client / server mode):

- a `goinfer` on a GPU machine that runs in client mode  
- a `goinfer` on a machine in a data‑center (static IP) that runs in server mode  
- the client `goinfer` connects to the server `goinfer` (here, the server is the backend of a web app)  
- the user sends their inference request to the backend (data‑center) which forwards it to the client `goinfer`  
- we could imagine installing a client `goinfer` on every computer with a good GPU, and the server `goinfer` that forwards inference requests to the connected client `goinfer` according to the requested model

### Low Priority

- Comprehensive **web admin** (monitoring, download/delete `.gguf`, edit config, restart, `git pull` + rebuild `llama.cpp`, remote shell, upgrade Linux, reboot the machine, and other SysAdmin tasks)

> **Contribute** – If you’re interested in any of the above, open an issue or submit a PR :)

### Nice to have

Some inspiration to extend the Goinfer stack:

- [`compose.yml`](./compose.yml) with something like [github.com/j4ys0n/local-ai-stack](https://github.com/j4ys0n/local-ai-stack) and [github.com/LLemonStack/llemonstack](https://github.com/LLemonStack/llemonstack)
- Multi-step AI agents like [github.com/synw/agent-smith](https://github.com/synw/agent-smith), [n8n](https://n8n.io/), [flowiseai](https://flowiseai.com/), tools from [github.com/HKUDS](https://github.com/HKUDS)
- WebUI [github.com/oobabooga/text-generation-webui](https://github.com/oobabooga/text-generation-webui), [github.com/danny-avila/LibreChat](https://github.com/danny-avila/LibreChat), [github.com/JShollaj/Awesome-LLM-Web-UI](https://github.com/JShollaj/Awesome-LLM-Web-UI)
- Vector Database and Vector Search Engine [github.com/qdrant/qdrant](https://github.com/qdrant/qdrant)
- Convert an webpage (URL) into clean markdown or structured data [github.com/firecrawl/firecrawl](https://github.com/firecrawl/firecrawl) [github.com/unclecode/crawl4ai](https://github.com/unclecode/crawl4ai) [github.com/browser-use/browser-use](https://github.com/browser-use/browser-use)
- [github.com/BerriAI/litellm](https://github.com/BerriAI/litellm) + [github.com/langfuse/langfuse](https://github.com/langfuse/langfuse)

## Contributing

1. Fork the repository.
2. Create a feature branch (`git checkout -b your‑feature`)
3. Run the test suite: `go test ./...` (more tests are welcome)
4. Ensure code is formatted and linted with `golangci-lint-v2 run --fix`
5. Submit a PR with a clear description and reference any related issue

Feel free to open discussions for design ideas/decisions.

## License

- **License:** MIT – see [`LICENSE`](./LICENSE) file.
- **Dependencies:**
  - [llama.cpp](https://github.com/ggml-org/llama.cpp) – Apache‑2.0
  - [llama‑swap](https://github.com/LM4eu/llama-swap) – MIT

## Acknowledgements

Special thanks to:

- [Georgi Gerganov](https://github.com/ggerganov) for releasing and improving [llama.cpp](https://en.wikipedia.org/wiki/Llama.cpp) in 2023 so we could freely play with Local LLM.
- [Benson Wong](https://github.com/mostlygeek) for maintaining [llama‑swap](https://github.com/mostlygeek/llama-swap) with clean and well‑documented code.
- the open‑source community that makes GPU‑based LLM inference possible on commodity hardware.

## See also

Some other active local-LLM proxies:

- [github/mostlygeek/llama-swap](https://github.com/mostlygeek/llama-swap)
- [github/inference-gateway/inference-gateway](https://github.com/inference-gateway/inference-gateway)
- [github/thushan/olla](https://github.com/thushan/olla)
- [github/lordmathis/llamactl](https://github.com/lordmathis/llamactl)
- [github/mostlygeek/llama‑swap](https://github.com/mostlygeek/llama-swap)
- [github/thushan/olla](https://github.com/thushan/olla)
- [github/x5iu/llm-proxy](https://github.com/x5iu/llm-proxy) (rust)
- [github/llm-proxy/llm-proxy](https://github.com/llm-proxy/llm-proxy) (python, inactive)
- [github/bluewave-labs/langroute](https://github.com/bluewave-labs/langroute)
- [github/codelion/optillm](https://github.com/codelion/optillm)

Compared to alternatives, we like [llama-swap](https://github.com/mostlygeek/llama-swap) for its readable source code and because its author contributes regularly. So we integrated it into Goinfer to handle communication with `llama-server` (or other compatible forks as [ik_llama.cpp](https://github.com/ikawrakow/ik_llama.cpp/)). We also like [llamactl](https://github.com/lordmathis/llamactl) ;-)

**Enjoy remote GPU inference with Goinfer!** 🚀

*If you have questions, need help setting up your first client/server pair, or want to discuss future features, open an issue or ping us on the repo’s discussion board.*
