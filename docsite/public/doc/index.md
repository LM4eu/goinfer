# Goinfer

Inference api server for local gguf language models. Based on [llama.cpp](https://github.com/ggml-org/llama.cpp) and [llama-swap](https://github.com/mostlygeek/llama-swap).

- **Multi models**: switch between models at runtime
- **Inference queries**: http api and websockets support

## Quickstart

Download a binary from the releases section (Linux only)

## Local usage with a gui

Generate your `goinfer.yml` config file,
providing the path to your models directory, 
where the `*.gguf` models are stored.
`goinfer` will also parse the subfolders, so you keep organizing your models within a folders tree.

```bash
MODELS_DIR=/path/to/my/models go run . -gen-gi-cfg -debug
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

## Api server usage

Generate a config file, providing the path to your models directory, 
where the `*.gguf` models are stored.
`goinfer` will also parse the subfolders,
so you keep organizing your models within a folders tree.

```bash
MODELS_DIR=/path/to/my/models ./goinfer -gen-gi-cfg
```

You can also provide multiple paths separated by `:` as the following:

```bash
MODELS_DIR=/path1:/path2:/path3
```

Note: the `-gen-gi-cfg` flag also generates your random API keys in the config file

Create a tasks directory, or edit the config file to provide a path to an existing one:

```bash
mkdir tasks
```

Run the server:

```bash
./goinfer
```

No gui is available, only the api