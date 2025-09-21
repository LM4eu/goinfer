# Goinfer

**Inference proxy for local LLMs** – run multiple `*.gguf` models on any machine, expose them through a secure HTTP‑API and optionally forward requests from a data‑center to idle GPUs in your home.

Built on top of **[llama.cpp](https://github.com/ggml-org/llama.cpp)** and **[llama‑swap](https://github.com/mostlygeek/llama-swap)**.

**TL;DR** – Deploy a **client** on a GPU‑rich desktop, a **server** on a machine with a static IP (or DNS), and let the server forward inference requests to the client. No VPN, no port‑forwarding, end‑to‑end encryption.

## Problem: Remote access to home-hosted LLM

Local‑LLM enthusiasts often hit a wall when they try to expose a model to the Internet:

- **Security** – exposing a raw `llama-server` or `ollama` instance can [leak the GPU to anyone](https://reddit.com/r/LocalLLaMA/comments/1nlpx3p).
- **Network topology** – most home routers block inbound connections, so the GPU machine can’t be *reached* from outside and home IP changes.
- **Privacy** – using third‑party inference services defeats the purpose of running models locally.

Existing tools (`llama‑swap`, `olla`, VPNs, WireGuard, etc.) either require inbound ports, complex network plumbing, or a custom client on every device.

**Goinfer** solves these issues by flipping the connection direction: the GPU‑rich **client** (home) *initiates* a secure outbound connection to a **server** with a static IP. The server then acts as a public façade, forwarding inference requests back to the client (home-hosted LLM).

## Key Features

Category            | Feature
--------------------|----------
**Model handling**  | Load multiple `*.gguf` models, switch at runtime, change any inference parameter
**API**             | OpenAI‑compatible HTTP API `/v1/`, streaming responses, Custom `/goinfer` API
**Security**        | Per‑role API keys (`admin`, `user`), CORS control
**Robustness**      | Independent of ISP‑provided IP, graceful reconnects
**Web UI**          | UI via [Infergui](https://github.com/synw/infergui)
**Admin control**   | Remote monitoring, delete/upload new GGUF files, reload config, `git pull llama.cpp`, re‑compile
**Home-hosted LLM** | Run Goinfer on your GPU desktop and another Goinfer in a data‑center (static IP/DNS)

## Client/Server Mode

    +-------------------+          +-------------------+          +-------------------+
    |  GPU‑rich desktop |  <--->   |   Goinfer Server  |  <--->   |   End‑user / App  |
    |   (goinfer client)           | (static IP/DNS)   |          |   (browser, API)  |
    +-------------------+          +-------------------+          +-------------------+

    Client  → initiates outbound TLS connection (HTTPS/HTTP3) → Server
    Server  → receives public requests, forwards them over the same channel → Client → llama‑server → model inference → response back through the tunnel.

*No inbound ports are opened on the client side.*

## Build

- Go 1.25+
- `llama.cpp` see [`scripts/llama-build-cuda.sh`](./scripts/llama-build-cuda.sh)
- One or more `*.gguf` model files.

```bash
git clone https://github.com/mostlygeek/llama-swap
git clone https://github.com/LM4eu/goinfer
git clone https://github.com/synw/infergui

cd infergui
npm run build
mv dist ../goinfer/go/infer

cd ../goinfer/go
go build -o goinfer .
```

## First Run

```bash
# generate a default configuration file
export GI_MODELS_DIR=/path/to/my/models
export GI_LLAMA_EXE=/path/to/my/llama-server
./goinfer -gen-gi-cfg

# and go
./goinfer -no-api-key
```

Goinfer will listen on the ports defined in the config. Default ports:

- `:5143` for the Web UI
- `:2222` for the OpenAI‑compatible API

```sh
curl -X POST localhost:2222/v1/chat/completions  \
  -H "Authorization: Bearer $GI_API_KEY_ADMIN"   \
  -H "Content-Type: application/json"            \
  -d '{
        "model": "default",
        "messages": [{"role":"user","content":"Hello, world!"}],
        "stream": false
      }'
```

## Configuration files

Generates `goinfer.yml` and `llama-swap.yml` with:

```bash
# optional
export GI_MODELS_DIR=/path/to/my/models
export GI_LLAMA_EXE=/path/to/my/llama-server
export GI_HOST=0.0.0.0
export GI_ORIGINS=
export GI_API_KEY_ADMIN="PLEASE SET SECURE API KEY"
export GI_API_KEY_USER="PLEASE SET SECURE API KEY"

cd goinfer/go

go run . -gen-gi-cfg  # goinfer.yml
go run . -gen-px-cfg  # llama-swap.yml
```

### API keys

The flag `-gen-gi-cfg` also generates two random API keys in the `goinfer.yml` config file.
This flag can be combined with:

- `-debug` to use debuq API key (only during the dev cycle)

- `-no-api-key` to avoid generating any API keys

### `GI_MODELS_DIR`

`GI_MODELS_DIR` is the root path where your models are stored.
`goinfer` will search `*.gguf` files within all `GI_MODELS_DIR` sub-folders.
So you can organize your models within a folders tree.

```bash
GI_MODELS_DIR=/path/to/my/models ./goinfer -gen-gi-cfg
```

You can also provide multiple paths separated by `:` as the following:

```bash
GI_MODELS_DIR=/path1:/path2:/path3
```

## Main `goinfer.yml`

```yaml
# Goinfer recursively search GGUF files in one or multiple folders separated by ':'
# List your GGUF dirs with `locate .gguf | sed -e 's,/[^/]*$,,' | uniq`
models_dir: /home/me/models 

server:
  api_key:
    # ⚠️ Set your API keys, can be 64‑hex‑digit (32‑byte) 🚨
    # Generate with `./goinfer -gen-gi-cfg`
    admin: "PLEASE SET ADMIN API KEY"
    user:  "PLEASE SET USER API KEY"
  origins:   # CORS whitelist
    - "https://my‑frontend.example.com"
    - "http://localhost"
  listen:
    # format:  <address>: <comma‑separated list of enabled services>
    # <address> can be <ip|host>:<port> or simply :<port> when <host> is localhost
    ":5143": webui,models           # UI + model list
    ":2222": openai,goinfer         # OpenAI‑compatible API + raw goinfer endpoint
    ":5555": llama-swap proxy       # optional llama‑swap proxy

llama:
  exe: /home/me/llama.cpp/build/bin/llama-server
  args:
    # common args used for every model
    common: "--props --no-webui --no-warmup"
    # extra args for the goinfer endpoint (Jinja templating)
    goinfer: "--jinja --chat-template-file template.jinja"
```

- **API keys** – Never commit them. Use env. vars `GI_API_KEY_ADMIN` `GI_API_KEY_USER` or a secrets manager in production.
- **Origins** – Set to the domains you’ll be calling the server from (including `localhost` for testing).
- **Ports** – Adjust as needed; make sure the firewall on the server allows them.

## Proxy `llama‑swap.yml`

Official documentation see: [github.com/mostlygeek/llama-swap/wiki/Configuration](https://github.com/mostlygeek/llama-swap/wiki/Configuration)

```yaml
logLevel: info            # debug, info, warn, error
healthCheckTimeout: 500   # seconds to wait for a model to become ready
metricsMaxInMemory: 1000  # maximum number of metrics to keep in memory
startPort: 6000           # first ${PORT} incremented for each model

macros:  # macros to reduce common conf settings
  "cmd-openai": "./llama-server --port ${PORT} --props --no-webui --no-warmup"
  "cmd-goinfer": "./llama-server --port ${PORT} --props --no-webui --no-warmup --jinja --chat-template-file template.jinja"

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

## Server / Client mode

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

```bash
curl -X POST https://my-goinfer-server.example.com/v1/chat/completions \
  -H "Authorization: Bearer <user-api-key>" \
  -H "Content-Type: application/json" \
  -d '{
        "model":"Qwen2.5-1.5B-Instruct-Q4_K_M",
        "messages":[{"role":"user","content":"Say hello in French"}],
        "stream":false
      }'
```

You should receive a JSON response generated by the model running on your GPU rig.

## API Endpoints

Service      | Path                   | Method | Description
-------------|------------------------|--------|------------
models       | `/models`              | GET    | List GGUF files currently present on the file system
proxy/openai | `/v1/chat/completions` | POST   | OpenAI‑compatible chat endpoint
proxy        | `/v1/models`           | GET    | List models from configuration
proxy        | `/v1/*`                | POST   | Classic completion endpoint
goinfer      | `/goinfer`             | POST   | Custom inference API
webui        | `/`                    | GET    | **Infergui**, static assets in `go/infer/dist`

All endpoints require an `Authorization: Bearer <API_KEY>` header. The `admin` key grants full access to the admin routes.

## Developer info

- **Config‑driven** setup with YAML files and environment variable overrides (`Cfg` defined in [`conf.go`](go/conf/conf.go:20)).
- Automatic **model discovery** in a configurable directory (`Search` in [`models.go`](go/conf/models.go:20)).
- Graceful shutdown handling (`handleShutdown` in [`goinfer.go`](go/goinfer.go:279)).
- API‑key authentication per service (`configureAPIKeyAuth` in [`router.go`](go/infer/router.go:119)).
- Comprehensive error handling with the `gie` package (`HandleError` in [`http.go`](go/gie/http.go:14)).

## History & Roadmap

### Origin

Goinfer has been initiated in March 2023 for two needs:

1. to swap engine and model at runtime, something that didn’t exist back then
2. to infer pre-configured templated prompts

This second point has been moved to the new project [github.com/synw/agent-smith](https://github.com/synw/agent-smith) with more templated prompts in [github.com/synw/agent-smith-plugins](https://github.com/synw/agent-smith-plugins).

### New needs

Today the needs have evolved. We need most right now is a proxy that can act as a secure intermediary between a **client (frontend/CLI)** and **a inference engine (local/cloud)** with these these constrains:

Client   | Server       | Constraint
---------|--------------|-----------
Frontend | OpenRouter   | Intermediate proxy required to manage the OpenRouter key without exposing it on the frontend
Any      | Home GPU rig | Access to another home GPU rig that forbids external TCP connections

### High Priority (✅ in progress)

1. Complete integration of `llama‑swap`(e.g.: swap model in RooCode)

2. Specific usage of `llama‑swap` API for `agent-smith` requirements (no default Jinja template). The hidden models for `agent-smith` are prefixed with `GI_`.

### Medium Priority

Two `goinfer` instances (client / server mode):

- a `goinfer` on a GPU machine that runs in client mode  
- a `goinfer` on a machine in a data‑center (static IP) that runs in server mode  
- the client `goinfer` connects to the server `goinfer` (here, the server is the backend of a web app)  
- the user sends their inference request to the backend (data‑center) which forwards it to the client `goinfer`  
- we could imagine installing a client `goinfer` on every computer with a good GPU, and the server `goinfer` that forwards inference requests to the connected client `goinfer` according to the requested model

### Low Priority

- `/goinfer` endpoint for full inference parameters control

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
  - [llama‑swap](https://github.com/mostlygeek/llama-swap) – MIT
  - [Infergui](https://github.com/synw/infergui) – MIT

## Acknowledgements

Special thanks to:

- [Georgi Gerganov](https://github.com/ggerganov) for releasing and improving [llama.cpp](https://en.wikipedia.org/wiki/Llama.cpp) in 2023 so we could freely play with Local LLM.
- [Benson Wong](https://github.com/mostlygeek) for maintaining [llama‑swap](https://github.com/mostlygeek/llama-swap) with clean and well‑documented code.
- the open‑source community that makes GPU‑based LLM inference possible on commodity hardware.

## See also

Some LLM Proxies:

- [github.com/mostlygeek/llama-swap](https://github.com/mostlygeek/llama-swap)
- [github.com/inference-gateway/inference-gateway](https://github.com/inference-gateway/inference-gateway)
- [github.com/thushan/olla](https://github.com/thushan/olla)

Compared to alternatives, we like [llama-swap](https://github.com/mostlygeek/llama-swap) for its readable source code and because its author contributes regularly. So we integrated it into Goinfer to handle communication with `llama-server` (or other inference engines compatible with the OpenAI API).

**Enjoy remote GPU inference with Goinfer!** 🚀

*If you have questions, need help setting up your first client/server pair, or want to discuss future features, open an issue or ping us on the repo’s discussion board.*
