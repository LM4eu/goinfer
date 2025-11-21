# llama-swap fork

This is a fork of [github/mostlygeek/llama-swap](https://github.com/mostlygeek/llama-swap)
for the project [github/LM4eu/goinfer](https://github.com/LM4eu/goinfer).

## background

Back in 2023, [Goinfer](https://github.com/LM4eu/goinfer)
was an early local LLM proxy swapping models and supporting
Ollama, Llamacpp, and KoboldCpp. To simplify the maintenance,
we decided in August 2025 to replace our process management with
another well-maintained project.

As we do not use Ollama / KoboldCpp any more,
we integrated [llama-swap](https://github.com/mostlygeek/llama-swap)
into Goinfer to handle communication with `llama-server`.

## issues with github/mostlygeek/llama-swap

1. The command `go get github.com/mostlygeek/llama-swap@v123` fails
   because the version numbering `v123` does not conform the Go standard `v1.2.3`.
   The workaround is to use `go get github.com/mostlygeek/llama-swap@main`
   that sets `v0.0.0-20250925224418-bab7d1f3968a` in `go.mod`.

2. Importing llama-swap using this workaround is not enough
   because the compilation requires the folder `proxy/ui_dist`
   that does not exist within the source code.

   ```go
   //go:embed ui_dist
   var reactStaticFS embed.FS
   ```

   The second workaround is to clone llama-swap and use a `go.work` file.

3. At LM4eu, we want to use the web UI of the underlying inference engine (e.g. llama.cpp).
   But the current llama-swap always require the model name within the client request (JSON).
   This is not possible to access a web page.

## changes

1. Use version `v0.0.123` compatible with Go expectations.
2. Add a minimalist `proxy/ui_dist/index.html`.
3. When no model is specified in the request, llama-swap defaults to the running model (first found).

## roadmap

We will adapt to the upstream project evolutions, while minimizing our patches.

But we may need to add more patches that may pollute the upstream project.

So we prefer to see if the project [github/LM4eu/goinfer](https://github.com/LM4eu/goinfer)
is successful. In that case we will discuss how to integrate our changes into the upstream project.

## merci

Special thanks to [Benson Wong](https://github.com/mostlygeek)
for maintaining [llama-swap](https://github.com/mostlygeek/llama-swap)
with clean and well-documented code.

Compared to some alternatives, we enjoy the readable source code of
[llama-swap](https://github.com/mostlygeek/llama-swap).
We also appreciate its author, [Benson Wong](https://github.com/mostlygeek),
regularly improves the code.
