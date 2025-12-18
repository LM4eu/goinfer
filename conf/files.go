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

	"github.com/LM4eu/goinfer/gie"
)

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
func getNameAndFlags(root, path string) (name, flags_ string) {
	truncated, flags := extractFlags(path)
	name = beautifyModelName(root, truncated)
	return name, flags
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

// extractFlags returns the truncated path and the llama-server flags from a file path.
// It first checks for a companion ".sh" file; if present, its contents are used as flags.
// Otherwise, it parses flags encoded in the filename after an '&' delimiter.
// Returns the truncated path (without extension) and a space-separated flag string.
//

func extractFlags(path string) (truncated, flags_ string) {
	truncated = strings.TrimSuffix(path, ".gguf")

	// Huge GGUF are spilt into smaller files ending with -00001-of-00003.gguf
	pos := strings.LastIndex(truncated, "-00001-of-")
	if pos > 0 {
		truncated = truncated[:pos]
	}

	// 1. Is there a file containing the command line arguments?
	shell := filepath.Clean(truncated + ".sh")
	args, err := os.ReadFile(shell)
	if err == nil {
		flags := oneLine(args)
		slog.Info("Found", "flags", flags, "from file", shell)
		return truncated, flags
	}

	// 2. Are there flags encoded within the filename?
	// Find the position of the last '/' (directory separator) and then locate the first '&' after that.
	slash := max(strings.LastIndexByte(truncated, '/'), 0)
	amp := strings.IndexByte(truncated[slash:], '&')
	if amp < 0 {
		return truncated, ""
	}
	pos = slash + amp

	var flags []string

	// Slice after the first '&' to avoid an empty first element.
	for f := range strings.SplitSeq(truncated[pos+1:], "&") {
		key, value, ok := strings.Cut(f, "=")
		if ok {
			key = "-" + key
			flags = append(flags, key, value)
		}
	}

	slog.Info("Found", "flags", flags, "from filename", truncated)
	return truncated[:pos], strings.Join(flags, " ")
}

// oneLine converts the `.sh` file into a single space-separated string,
// removing trailing backslashes, trimming whitespace, ignoring empty lines or comments.
func oneLine(input []byte) string {
	keep := make([]byte, 0, len(input))

	for line := range bytes.SplitSeq(input, []byte("\n")) {
		// Remove trailing backslash
		if bytes.HasSuffix(line, []byte("\\")) {
			line = line[:len(line)-1]
		}
		// Remove leading/trailing whitespace
		line = bytes.TrimSpace(line)
		// Skip blank lines
		if len(line) == 0 {
			continue
		}
		// Convert the byte slice to a string before appending.
		if line[0] == '-' {
			keep = append(keep, line...)
			keep = append(keep, ' ')
		}
	}

	return string(keep)
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
			return gie.New(gie.ConfigErr, "No *.gguf files found", "dir", cfg.ModelsDir)
		}
		slog.Warn("No model configured => Restart Goinfer to refresh llama-swap.yml", "models", n)
		return nil
	}

	for i := range cfg.Swap.Models {
		var previous string
		for arg := range strings.SplitSeq(cfg.Swap.Models[i].Cmd, " ") {
			if previous == "-m" || previous == "--model" {
				modelFile := arg // the argument after -m|--model is the GUFF file
				_, err := verify(modelFile)
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
func verify(path string) (int64, error) {
	cleaned := filepath.Clean(path)
	if cleaned != path {
		slog.Warn("Malformed", "current", path, "better", cleaned)
		path = cleaned
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		slog.Warn("Model file does not exist", "path", path)
		return 0, err
	}

	// is empty?
	size := info.Size()
	if size < 1000 {
		return 0, gie.New(gie.ConfigErr, "Model file is empty (or too small)", "path", path, "size", size)
	}

	// Check if the file is readable
	file, err := os.Open(path)
	if err != nil {
		slog.Warn("Model file is not readable", "path", path)
		return 0, err
	}

	err = file.Close()
	if err != nil {
		slog.Warn("Model file fails closing", "path", path)
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
		return 0, gie.New(gie.ConfigErr, "Model file is part of a series, but only the first one is referenced, file="+path)
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
