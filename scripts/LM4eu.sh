#!/usr/bin/env bash

# this script applies the LM4eu patches to llama-swap

# Safe bash
set -e                   # stop the script if any command returns a non-zero status
set -u                   # unset variable is an error => exit
set -o pipefail          # pipeline fails if any of its components fails
set -o noclobber         # prevent accidental file overwriting with > redirection
shopt -s inherit_errexit # apply these restrictions to $(command substitution)

# Color logs
log() { set +x; echo >&2 -e "\033[34m$(date +%H:%M)\033[m \033[32m" "$@" "\033[m"; }
err() { set +x; echo >&2 -e "\033[34m$(date +%H:%M)\033[m \033[31m" "$@" "\033[m"; }

# print the script line number if something goes wrong
set -E
trap 's=$?; err "exit status=$? at ${BASH_SOURCE[0]}:$LINENO" >&2; exit $s' ERR

dir="${BASH_SOURCE[0]%/*}"
cd "$dir/.."
pwd

(
    log "switch to latest upstream/main"
    set -x
    git status
    git fetch upstream --prune -t
    git reset --hard   upstream/main
    git switch -C main upstream/main
    git status
)

version="$(git tag --list --sort -v:refname | grep '[0-9]' -m1)"
digits=${version//[^0-9]/}  # if version=v166 => tag=v0.166.0
patch="${patch:-0}"
branch="${branch:-lm4-$digits.$patch}"
tag="${tag:-v0.$digits.$patch}"

(
    log "found version $version  ->  create branch $branch"
    set -x
    git switch --force-create "$branch" # "origin"
    git status
)

log "replace README"

cat >| README.md <<'EOL'
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
EOL

(
    set -x
    git status
    git commit -m 'README: explain the LM4eu fork' README.md
    git status
)

(
    log "add a basic proxy/ui_dist/index.html"
    set -x
    git status
    mkdir -pv proxy/ui_dist
)

echo >| proxy/ui_dist/index.html '<html>
<head><title>llama-swap - Missing UI</title></head>
<body>
<h1>llama-swap - Missing UI</h1>
<p>To build the web static assets:</p>
<tt>
cd llama-swap/ui
npm ci --prefer-offline --no-audit --no-fund
npm run build
</tt>
<p>Then check the folder <code>proxy/ui_dist</code></p>
</body>
</html>'

(
    set -x
    git add -f proxy/ui_dist/index.html
    git commit -m 'proxy: add default index.html to allow another Go code to import llama-swap'
)

(
    log "replace mostlygeek/llama-swap -> LM4eu/llama-swap"
    set -x
    git status
    sed -i -e 's,module github.com/mostlygeek/llama-swap,module github.com/LM4eu/llama-swap,' go.mod
    git add go.mod
    find -name "*.go" -exec sed -i -e 's,"github.com/mostlygeek/llama-swap,"github.com/LM4eu/llama-swap,' {} + -exec git add {} +
    git status
    go run . -h 2>/dev/null # smoke test
    git commit -m "fork mostlygeek -> LM4eu"
)

(
    log "add missing MacroList.MarshalYAML() in proxy/config/config.go"
    set -x
    git status
    echo >> proxy/config/config.go "
// MarshalYAML cannot guarantee the order because it returns map[string]any
func (ml MacroList) MarshalYAML() (any, error) {
	return ml.ToMap(), nil
}
"
    go run . -h 2>/dev/null # smoke test
    git commit -m 'config: add missing MacroList.MarshalYAML()' proxy/config/config.go
)

(
    log "add ProxyToFirstRunningProcess in proxymanager.go"
    set -x
    git status
    echo >> proxy/proxymanager.go '
// ProxyToFirstRunningProcess forwards the request to the any running process (llama-server)
func (pm *ProxyManager) ProxyToFirstRunningProcess(c *gin.Context) {
	for _, processGroup := range pm.processGroups {
		for _, process := range processGroup.processes {
			if process.CurrentState() == StateReady {
				process.ProxyRequest(c.Writer, c.Request)
				return
			}
		}
	}
	pm.sendErrorResponse(c, http.StatusInternalServerError, "No model currently running. Please select a model.")
}
'
    go run . -h 2>/dev/null # smoke test
    git commit -m 'proxy: forward the request to the any running process

This is useful when model is not specified' proxy/proxymanager.go
)

(
    log "export seven endpoint handlers"
    set -x
    git status
    sed -i 's/\<proxyOAIHandler\>/ProxyOAIHandler/g;
            s/\<proxyOAIPostFormHandler\>/ProxyOAIPostFormHandler/g;
            s/\<listModelsHandler\>/ListModelsHandler/g;
            s/\<streamLogsHandler\>/StreamLogsHandler/g;
            s/\<unloadAllModelsHandler\>/UnloadAllModelsHandler/g;
            s/\<listRunningProcessesHandler\>/ListRunningProcessesHandler/g;
    '       proxy/proxymanager.go proxy/proxymanager_loghandlers.go
    git add proxy/proxymanager.go proxy/proxymanager_loghandlers.go
    go run . -h 2>/dev/null # smoke test
    git commit -m 'proxy: export seven endpoint handlers

Export these endpoint handlers (Capitalize the initial)

1. proxyOAIHandler
2. proxyOAIPostFormHandler
3. listModelsHandler
4. streamLogsHandler
6. unloadAllModelsHandler
7. listRunningProcessesHandler

This change allows using the llama-swap API
as a library with direct function call
and avoid the HTTP request/response overhead.
This reduces Goinfer latency and code complexity.'
)

old='"gopkg.in/yaml.v3"'
new='"go.yaml.in/yaml/v4"'

(
    log "replace gopkg.in/yaml.v3 -> go.yaml.in/yaml/v4"
    set -x
    git status
    grep --include=*.go -RlF "$old" . | xargs bash -xc 'set -- "$0" "$@"; sed -i -e '"'s|$old|$new|g'"' "$@" ; git add "$@"'
    git status
    git commit -m "replace $old -> $new"
)

(
    log "refresh go.mod"
    set -x
    git status
    rm go.sum go.mod
    go mod init github.com/LM4eu/llama-swap
    go mod tidy
    go run . -h 2>/dev/null # smoke test
    git status
    git commit -m "go.mod: refresh + replace $old -> $new" go.sum go.mod
)

(
    log "add LM4eu.sh and LM4eu.patch"
    set -x
    git status
    # we keep the original LM4eu.sh LM4eu.patch as untracked in another folder
    # because Git will remove them when switching to another branch
    pwd
    cp -fv "$dir"/LM4eu.sh "$dir"/LM4eu.patch scripts/
    chmod +x scripts/LM4eu.sh
    git add  scripts/LM4eu.sh scripts/LM4eu.patch
    git commit -m 'scripts: add LM4eu.sh and LM4eu.patch'
)

(
    log "found upstream tag=$version  ->  create tag=$tag"
    set -x
    git tag -f "$tag"
)

(
    log "merge branch $branch into lm4 (lm4 is the default branch name of the LM4eu fork)"
    set -x
    git fetch origin
    #git reset --hard  origin/lm4
    git switch -C lm4 origin/lm4    
    #git branch -u origin/lm4 lm4    # set upstream tracking (already done by `switch -C`)
    git status
    git merge -X theirs --rerere-autoupdate --verbose --stat --progress --autostash -m "Merge branch '$branch' into lm4" "$branch"
)

(
    log "provide the last 16 commits"
    set -x
    git log -16 --oneline --decorate --graph --date=short --ignore-space-change --ignore-blank-lines --find-copies-harder --follow .
    git status
)

log "
    success 😀
    please verify, when ready:

    git push
    git push origin "$tag"
"
