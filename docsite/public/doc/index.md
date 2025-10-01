# Goinfer

Inference proxy server for local large language models (LLM) using `*.gguf` files.
Goinfer is based on [llama.cpp](https://github.com/ggml-org/llama.cpp) and [llama-swap](https://github.com/mostlygeek/llama-swap).

- **Multi models**: switch between models at runtime
- **Inference queries**: HTTP API and streaming response support
- **Admin web UI**: [Infergui](https://github.com/synw/infergui)

## Quickstart

Download a binary from the releases section (Linux only)

## Local usage with a gui

Generate your `goinfer.yml` config file,
providing the path to your models directory,
where the `*.gguf` models are stored.
`goinfer` will also parse the subfolders, so you keep organizing your models within a folders tree.

```bash
GI_MODELS_DIR=/path/to/my/models go run . -gen -debug
```

Provide the path to your `*.gguf` models directory

This will create a `goinfer.yml` file

Create a tasks directory:

```bash
mkdir tasks
```

Run the server:

```bash
./goinfer -local
```

Open `http://localhost:5143` in a browser to get the gui

## API server usage

Generate a config file, providing the path to your models directory (where  reside).
`goinfer` will search `*.gguf` files within all subfolders.
So you can organize your models within a folders tree.

```bash
GI_MODELS_DIR=/path/to/my/models ./goinfer -gen
```

You can also provide multiple paths separated by `:` as the following:

```bash
GI_MODELS_DIR=/path1:/path2:/path3
```

Note: the `-gen` flag also generates your random API keys in the config file.

Run the server:

```bash
./goinfer
```
