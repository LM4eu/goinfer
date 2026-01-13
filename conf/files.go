// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"bytes"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/lynxai-team/garcon/gerr"
)

type Root struct {
	FS   fs.FS
	Path string
}

func NewRoot(path string) Root {
	return Root{os.DirFS(path), path}
}

func (r *Root) Open(name string) (fs.File, error) {
	return r.FS.Open(name)
}

func (r *Root) FullPath(relPath string) string {
	if relPath == "" {
		return ""
	}
	return filepath.Join(r.Path, relPath)
}

func (r *Root) RelativePath(fullPath string) (string, error) {
	relativePath, err := filepath.Rel(r.Path, fullPath)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(relativePath) {
		return "", gerr.New(gerr.Invalid, "not local", "full", fullPath, "root", r.Path)
	}
	return filepath.Clean(relativePath), err
}

// replaceDIR in flags by the current dir of he file.
// When using models like GPT OSS, we need to provide a grammar file.
// See: github.com/ggml-org/llama.cpp/discussions/15396#discussioncomment-14145537
// We want the possibility to keep both model and grammar files in the same folder.
// But we also want to be free to move that folder
// without having to update the path within tho command line arguments.
// Thus, we use $DIR as a placeholder for the folder.
func replaceDIR(path, flags string) string {
	return strings.ReplaceAll(flags, "$DIR", filepath.Dir(path))
}

// getNameAndFlags returns model name and llama-server flags.
func getNameAndFlags(root, path string) (name, flags, origin string) {
	truncated, flags, origin := extractFlags(path)
	name = beautifyModelName(root, truncated)
	return name, flags, origin
}

// beautifyModelName converts the first underscore in a model name to a slash.
func beautifyModelName(root, truncated string) string {
	name := filepath.Base(truncated)

	withGGUF := nameWithGGUF(name)
	if withGGUF != "" {
		return withGGUF
	}

	name = nameWithSlash(root, truncated, name)
	name = strings.TrimSuffix(name, "_")
	name = strings.TrimSuffix(name, "-GGUF")
	name = strings.Replace(name, "-GGUF_", ":", 1)
	name = strings.Replace(name, "-GGUF:", ":", 1)

	return name
}

// nameWithSlash converts the first underscore in a model name to a slash or use the folder name.
// If there is a dash before the 1st underscore (e.g. ggml-org_gpt...),
// consider a valid grp only if 3 or 4 chars between dash and underscore (e.g. org).
func nameWithSlash(root, truncated, name string) string {
	dash := -1 // position of the dash sign

	for i, char := range name {
		if isLimitReached(i, dash) {
			return nameWithDir(root, truncated, name)
		}

		switch {
		case char == '-': // dash
			if isWrongDash(i, dash) {
				return nameWithDir(root, truncated, name)
			}
			dash = i
		case char == '_': // underscore
			if isWrongUnderscore(i, dash) {
				return nameWithDir(root, truncated, name)
			}
			out := []byte(name)
			out[i] = '/'
			return string(out)
		case !unicode.IsLower(char):
			return nameWithDir(root, truncated, name)
		default:
		}
	}
	return nameWithDir(root, truncated, name)
}

func isLimitReached(i, dash int) bool {
	if i > 9 {
		if dash < 0 { // the limit is 9 letters without a dash
			return true
		}
		if i > 11 { // otherwise the limit is 10 letters + one dash
			return true
		}
	}
	return false
}

func isWrongDash(i, dash int) bool {
	if i < 4 {
		return true
	}
	if dash > -1 {
		return true
	}
	return false
}

func isWrongUnderscore(i, dash int) bool {
	if dash > 0 {
		n := i - dash // number of letters before the dash
		ok := n == 3 || n == 4
		if !ok {
			return true
		}
	}
	if i < 4 {
		return true
	}
	if i-dash < 3 {
		return true
	}
	return false
}

// nameWithGGUF detects files downloaded from HuggingFace (flag -hf).
// Patterns:
//   - cmd: -hf ggml-org/gpt-oss-120b-GGUF
//     in = ggml-org_gpt-oss-120b-GGUF_gpt-oss-120b-mxfp4
//     out = ggml-org/gpt-oss-120b
//   - cmd: -hf unsloth/Devstral-2-123B-Instruct-2512-GGUF:UD-Q4_K_XL
//     in = unsloth_Devstral-2-123B-Instruct-2512-GGUF_UD-Q4_K_XL_Devstral-2-123B-Instruct-2512-UD-Q4_K_XL
//     out: unsloth/Devstral-2-123B-Instruct-2512:UD-Q4_K_XL
func nameWithGGUF(name string) string {
	const gguf = "-GGUF_"
	pos := strings.Index(name, gguf)
	if pos <= 0 {
		return ""
	}

	// Expected Cut:
	// grp   = ggml-org     unsloth
	// model = gpt-oss-120b Devstral-2-123B-Instruct-2512
	grp, model, ok := strings.Cut(name[:pos], "_")
	if !ok {
		return ""
	}

	// search for the duplicated model name
	after := name[pos+len(gguf):]
	pos = strings.Index(after, model)
	if pos < 0 {
		return ""
	}

	name = grp + "/" + model
	if pos > 1 {
		quants := after[:pos-1]
		name += ":" + quants
	}
	return name
}

// nameWithDir prefixes the model name with its folder name.
// If there is a dash in the directory name (e.g. ggml-org/file.gguf),
// consider a valid grp only if 3 or 4 chars after the dash (e.g. org).
func nameWithDir(root, truncated, name string) string {
	dir := filepath.Dir(truncated)
	if len(dir) <= len(root) {
		return name
	}
	grp := filepath.Base(dir)
	dash := -1
	for i, char := range grp {
		switch {
		case i > 12:
			return name
		case char == '-':
			if i < 4 {
				return name
			}
			if dash > -1 {
				return name
			}
			dash = i // number of letters before the dash
		case !unicode.IsLower(char):
			return name
		default:
		}
	}
	if dash > 0 {
		n := len(grp) - dash // n = number of letters after the dash
		ok := n == 3 || n == 4
		if !ok {
			return name
		}
	}
	return grp + "/" + name
}

// extractModelNameAndFlags search for a llama-server command line
// and extract the model path (flag -m or --model) and the flags after -m|--model.
func extractModelNameAndFlags(root Root, shellPath string) (modelPath, flags []byte) {
	script, err := fs.ReadFile(root.FS, shellPath)
	if err != nil {
		return nil, nil
	}

	script = searchLlamaServer(script)
	if script == nil {
		return nil, nil
	}

	script = searchModelFlag(script)
	if script == nil {
		return nil, nil
	}

	// trim any leading whitespace character
	script = bytes.TrimSpace(script)
	pos := bytes.IndexAny(script, " \t")
	if pos < 0 {
		modelPath = script
	} else {
		modelPath = script[:pos]
		flags = oneLine(script[pos+1:])
	}

	slog.Info("Find shell", "root", root.Path, "file", shellPath, "model", modelPath, "flags", flags)

	return modelPath, flags
}

// searchLlamaServer searches for a llama-server command line.
func searchLlamaServer(script []byte) []byte {
	for {
		var before []byte
		var found bool
		before, script, found = bytes.Cut(script, []byte("/llama-server"))
		if !found {
			return nil
		}

		if len(script) == 0 {
			return nil
		}

		// "/llama-server" must be followed by a space
		justAfter := script[0]
		if justAfter != ' ' && justAfter != '\t' {
			continue
		}

		if len(before) == 0 {
			return script[1:] // skip the justAfter (a space)
		}

		// rewind to the beginning of the line
		pos := bytes.LastIndexByte(before, '\n')
		if pos >= 0 {
			before = before[pos+1:]
		}

		// skip commented lines except shebang (#!)
		if before[0] != '#' || (pos < 0 && len(before) > 1 && before[1] == '!') {
			return script[1:] // skip the justAfter (a space)
		}
	}
}

// searchModelFlag searches for the flag: --model or -m.
//
//nolint:revive // function is easy to understand
func searchModelFlag(script []byte) []byte {
	for {
		before, after, found := bytes.Cut(script, []byte("--model"))
		if !found {
			before, after, found = bytes.Cut(script, []byte("-m"))
			if !found {
				return nil
			}
		}
		script = after

		if len(script) == 0 {
			return nil
		}

		// --model must be followed by a space
		justAfter := script[0]
		if justAfter != ' ' && justAfter != '\t' {
			continue
		}

		if len(before) == 0 {
			return script[1:] // skip the justAfter (a space)
		}

		// --model must be preceded by a space
		justBefore := before[len(before)-1]
		if justBefore != ' ' && justBefore != '\t' {
			continue
		}

		// rewind to the beginning of the line
		pos := bytes.LastIndexByte(before, '\n')
		if pos >= 0 {
			before = before[pos+1:]
		}

		// skip commented lines
		if before[0] != '#' {
			return script[1:] // skip the justAfter (a space)
		}
	}
}

// extractFlags returns the truncated path and the llama-server flags from a file path.
// It first checks for a companion ".sh" file; if present, its contents are used as flags.
// Otherwise, it parses flags encoded in the filename after an '&' delimiter.
// Returns the truncated path (without extension) and a space-separated flag string.
func extractFlags(path string) (truncated, flags, origin string) {
	truncated = strings.TrimSuffix(path, ".gguf")

	// Huge GGUF are spilt into smaller files ending with -00001-of-00003.gguf
	pos := strings.LastIndex(truncated, "-00001-of-")
	if pos > 0 {
		truncated = truncated[:pos]
	}

	// 1. Is there a file containing the command line arguments?
	shellPath := filepath.Clean(truncated + ".sh")
	shellData, err := os.ReadFile(shellPath)
	if err == nil {
		flags = string(oneLine(shellData))
		slog.Info("Found from", "script", shellPath, "flags", flags)
		return truncated, flags, shellPath
	}

	// 2. Are there flags encoded within the filename?
	// Find the position of the last '/' (directory separator) and then locate the first '&' after that.
	slash := max(strings.LastIndexByte(truncated, '/'), 0)
	amp := strings.IndexByte(truncated[slash:], '&')
	if amp < 0 {
		return truncated, "", ""
	}
	pos = slash + amp

	var args []string

	// Slice after the first '&' to avoid an empty first element.
	for f := range strings.SplitSeq(truncated[pos+1:], "&") {
		key, value, ok := strings.Cut(f, "=")
		if ok {
			key = "-" + key
			args = append(args, key, value)
		}
	}

	slog.Info("Found from", "filename", truncated, "flags", flags)
	return truncated[:pos], strings.Join(args, " "), "GGUF filename"
}

// oneLine converts the `.sh` file into a single space-separated string,
// removing trailing backslashes, trimming whitespace, ignoring empty lines or comments.
func oneLine(input []byte) []byte {
	keep := make([]byte, 0, len(input))

	for line := range bytes.SplitSeq(input, []byte("\n")) {
		if len(line) == 0 {
			continue // Skip empty lines
		}

		if line[len(line)-1] == '\\' {
			line = line[:len(line)-1] // Remove trailing backslash
		}

		// Skip blank lines, commented lines (keep only lines starting with a dash)
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '-' {
			continue
		}

		if len(keep) > 0 {
			keep = append(keep, ' ')
		}
		keep = append(keep, line...)
	}

	return keep
}

// ValidateSwap checks that the configuration contains at least one model file and
// that each model referenced in the swap configuration exists on disk.
// It logs warnings and errors as appropriate.
func (cfg *Cfg) ValidateSwap() error {
	if cfg.Swap == nil {
		return nil // nothing to validate
	}

	if len(cfg.Swap.Models) == 0 {
		n := len(cfg.getInfo())
		if n == 0 {
			return gerr.New(gerr.ConfigErr, "No *.gguf files found", "dir", cfg.ModelsDir)
		}
		slog.Warn("No model configured => Restart Goinfer to refresh llama-swap.yml", "models", n)
		return nil
	}

	for i := range cfg.Swap.Models {
		var previous string
		for arg := range strings.FieldsSeq(cfg.Swap.Models[i].Cmd) {
			if previous == "-m" || previous == "--model" {
				modelFile := arg // the argument after -m|--model is the GUFF file
				_, err := verify(NewRoot("/"), "."+modelFile)
				if err != nil {
					return err
				}
				break
			}
			previous = arg
		}
	}
	return nil
}

// verify that the given GUFF file is an existing,
// readable, and sufficiently large *.gguf file.
// It also normalizes the path and checks for series files.
func verify(root Root, path string) (int64, error) {
	cleaned := filepath.Clean(path)
	if cleaned != path {
		slog.Warn("Malformed", "current", path, "better", cleaned)
		path = cleaned
	}

	info, err := fs.Stat(root.FS, path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Model does not exist", "path", path)
		}
		return 0, err
	}

	// is empty?
	size := info.Size()
	if size < 1000 {
		return 0, gerr.New(gerr.ConfigErr, "Model file is empty (or too small)", "path", path, "size", size)
	}

	// Check if the file is readable
	file, err := root.FS.Open(path)
	if err != nil {
		slog.Warn("Model is not readable", "path", path)
		return 0, err
	}

	err = file.Close()
	if err != nil {
		slog.Warn("Model Close() fails", "path", path)
		return 0, err
	}

	// Huge GGUF are spilt into smaller files ending with -00001-of-00003.gguf
	// Keep only the first one, and ignore the others: -00002-of-00003.gguf
	pos := strings.LastIndex(path, "-of-")
	const first = "00001"
	if pos < len(first) {
		return size, nil // OK
	}

	if path[pos-len(first):pos] != first {
		slog.Debug("KO Model file is part of a series, but only the first one is referenced", "path", path)
		return 0, gerr.New(gerr.ConfigErr, "Model file is part of a series, but only the first one is referenced, file="+path)
	}

	slog.Debug("OK Model file is the first of a series", "path", path)
	return size, nil // OK
}

// DiscoverModelParentFolders returns a sorted slice
// of *top‑most* directories that contain *.gguf files.
// If no root paths are supplied, the function scans
// the typical locations: /mnt /var/opt /opt /home/username
//
// Example usage:
//
//	dirs := conf.DiscoverModelParentFolders() // []string{"/home/bob/models", "/mnt/models"}
//
// This Go function is a rewrite of the following bash command line:
//
//	p=; find /mnt /var/opt /opt "$HOME" -type f -name '*.gguf' -printf '%h\0' | sort -zu |
//	while IFS= read -rd '' d; do [[ $p && $d == "$p"/* ]] && continue; echo -n "$d:"; p=$d; done
//
// This bash command line discovers the parent folders of the GUFF files:
//   - find the files *.gguf in /mnt /var/opt /opt "$HOME" directories
//   - -printf their folders (%h) separated by nul character `\0`
//     (support folder names containing newline characters)
//   - sort them, -u to keep a unique copy of each folder (`z` = input is `\0` separated)
//   - while read xxx; do xxx; done  =>  keep the parent folders only
//   - echo $d: prints each parent folder separated by ":" (`-n` no newline)
func DiscoverModelParentFolders(roots ...string) []string {
	// default roots = /mnt /var/opt /opt /home/username
	if len(roots) == 0 {
		roots = []string{"/mnt", "/var/opt", "/opt"}
		home, _ := os.UserHomeDir()
		if home != "" {
			roots = append(roots, home)
		}
	}

	// collect all directories containing *.gguf files
	dirSet := map[string]struct{}{} // set for uniqueness
	for _, r := range roots {
		_ = filepath.Walk(r, func(path string, fi os.FileInfo, e error) error {
			if e == nil && !fi.IsDir() && strings.HasSuffix(path, ".gguf") {
				dirSet[filepath.Dir(path)] = struct{}{}
			}
			return nil
		})
	}

	// convert the set to a sorted slice
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	// drop sub‑directories that are already covered by a higher‑level entry
	parents := make([]string, 0, len(dirs))
	sep := string(os.PathSeparator)
	for _, d := range dirs {
		if len(parents) > 0 && strings.HasPrefix(d, parents[len(parents)-1]+sep) {
			continue // skip d = child of the previously kept directory
		}
		parents = append(parents, d)
	}
	return parents
}

// DiscoverModelsTree returns the GGUF files tree
// by walking the supplied roots (or a default set).
// It returns a `map[parentDir]map[childDir][]file` where:
//
//   - parentDir – shallowest directory that contains at least one *.gguf file.
//   - childDir – path relative to that parent (empty string for files directly inside the parent).
//   - file   – the basename of the *.gguf file (including the ".gguf" suffix).
//
// All filesystem errors are ignored:
// the function always returns whatever it could discover.
//
// Example (files on disk):
//
//	/home/bob/models/model1.gguf
//	/home/bob/models/subdir/model2.gguf
//	/mnt/models/model3.gguf
//
//	tree := DiscoverModelsTree()
//
//	// equivalent to:
//
//	tree = map[string]map[string][]string{
//	    "/home/bob/models": {
//	        "":         {"model1.gguf"},
//	        "subdir":   {"model2.gguf"},
//	    },
//	    "/mnt/models": {
//	        "":         {"model3.gguf"},
//	    },
//	}
func DiscoverModelsTree(roots ...string) map[string]map[string][]string {
	if len(roots) == 0 {
		// default roots = /mnt /var/opt /opt /home/$USER
		roots = []string{"/mnt", "/var/opt", "/opt"}
		home, _ := os.UserHomeDir()
		if home != "" {
			roots = append(roots, home)
		}
	}

	dirFiles := collectModelFiles(roots)

	return buildModelsTree(dirFiles)
}

// collectModelFiles walks each root and groups *.gguf files by the directory that holds them.
func collectModelFiles(roots []string) map[string][]string {
	dirFiles := map[string][]string{} // dir → []basename
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // ignore unreadable paths and directories
			}
			if strings.EqualFold(filepath.Ext(path), ".gguf") {
				dir := filepath.Dir(path)
				dirFiles[dir] = append(dirFiles[dir], filepath.Base(path))
			}
			return nil
		})
	}
	return dirFiles
}

// buildModelsTree turns the flat dir→files map into the hierarchical result.
func buildModelsTree(dirFiles map[string][]string) map[string]map[string][]string {
	// sort the directory keys so parents come first.
	dirs := make([]string, 0, len(dirFiles))
	for d := range dirFiles {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	// Collapse children under their parent directory.
	const sep = string(filepath.Separator)
	tree := map[string]map[string][]string{}
	var parent string // current parent directory

	for _, d := range dirs {
		files := dirFiles[d]
		sort.Strings(files) // deterministic order of file names

		// Is this a new top‑level parent?
		if parent == "" || !strings.HasPrefix(d, parent+sep) {
			parent = d
			tree[parent] = map[string][]string{"": files}
			continue
		}
		// d is a descendant of the current parent
		relativeDir := d[len(parent)+1:] // trim parent dir
		tree[parent][relativeDir] = files
	}
	return tree
}
