# Goinfer

Inference proxy server for local large language models (LLM) using `*.gguf` files.
Goinfer is based on [llama.cpp](https://github.com/ggml-org/llama.cpp) and [llama-swap](https://github.com/mostlygeek/llama-swap).

- **Multi models**: switch between models at runtime
- **Inference queries**: HTTP API and streaming response support

## Quickstart

Download a binary from the releases section (Linux only)

The following command generates
the `goinfer.yml` config file.

```bash
GI_MODELS_DIR=/path/to/my/models ./goinfer -gen -debug
```

Use `GI_MODELS_DIR` to provide the path to your models directory
(where the `*.gguf` models are stored).

You can also provide multiple paths separated by `:` as the following:

```bash
GI_MODELS_DIR=/path1:/path2:/path3
```

`goinfer` will search the model files within the sub-folders,
so you can keep organizing your models within a folders tree.

Note: the `-gen` flag also generates your random API keys in the config file.

Once you have your `goinfer.yml` file, run the server:

```bash
./goinfer
```
