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
	"strings"
	"unicode"

	"github.com/LM4eu/goinfer/gie"
)

// ModelInfo is used for the response of the /models endpoint, including:
// - command‑line flags found of file system
// - eventual error (if the model is missing or misconfigured).
type ModelInfo struct {
	Flags string `json:"flags,omitempty" yaml:"flags,omitempty"`
	Path  string `json:"path,omitempty"  yaml:"path,omitempty"`
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// ListModels returns the model names from the config and from the models_dir.
func (cfg *Cfg) ListModels() (map[string]ModelInfo, error) {
	info, err := cfg.search()
	if err != nil {
		slog.Debug("Search models", "err", err)
	}

	const notConfigured = "file present but not configured in llama-swap.yml"

	for name, mi := range info {
		if info[name].Error == "" {
			info[name] = ModelInfo{mi.Flags, mi.Path, notConfigured}
		}
	}

	for name := range cfg.Swap.Models {
		if len(name) > 3 && name[:3] == "GI_" && cfg.Swap.Models[name].Unlisted {
			continue // do not report models for /goinfer endpoint
		}

		mi, ok := info[name]
		if ok {
			if mi.Error == notConfigured {
				mi.Error = "" // OK: model is both present in FS and configured in llama-swap.yml
			}
		} else {
			cmd := strings.SplitN(cfg.Swap.Models[name].Cmd, "--model", 2)
			if len(cmd) > 0 {
				mi.Flags = cmd[0]
			}
			if len(cmd) > 1 {
				mi.Path = cmd[1]
			}
			mi.Error = "file absent but configured in llama-swap.yml"
		}
		info[name] = mi
	}

	return info, err
}

// search returns a slice of absolute file paths for all *.gguf model files
// found under the directories listed in cfg.ModelsDir (colon-separated).
// It walks each directory recursively, aggregates matching files,
// and returns any error encountered.
func (cfg *Cfg) search() (map[string]ModelInfo, error) {
	info := make(map[string]ModelInfo, len(cfg.Swap.Models)/2)

	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		err := add(info, strings.TrimSpace(root))
		if err != nil {
			slog.Debug("Searching model files", "root", root)
			return nil, err
		}
	}

	return info, nil
}

// add walks the given root directory and appends any valid *.gguf model file paths to the
// provided slice. It validates each file using validateFile and logs debug information.
func add(info map[string]ModelInfo, root string) error {
	return filepath.WalkDir(root, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			if dir == nil {
				return gie.Wrap(err, gie.NotFound, "filepath.WalkDir")
			}
			return gie.Wrap(err, gie.NotFound, "filepath.WalkDir path="+dir.Name())
		}

		if dir.IsDir() {
			return nil // => step into this directory
		}

		if !strings.HasSuffix(path, ".gguf") {
			return nil
		}

		err = validateFile(path)
		if err != nil {
			slog.Debug("Skip", "model", path)
			return nil //nolint:nilerr // "return nil" to skip this file
		}

		slog.Debug("Found", "model", path)

		name, flags := getNameAndFlags(root, path)
		mi := ModelInfo{flags, path, ""}
		if old, ok := info[name]; ok {
			slog.Warn("Duplicated models", "dir", root, "name", name, "old", old, "new", mi)
			mi.Error = "two files have same model name (must be unique)"
		}
		info[name] = mi

		return nil
	})
}

//nolint:gocritic,revive // return model name and llama-server flags
func getNameAndFlags(root, path string) (string, string) {
	truncated, flags := extractFlags(path)
	name := nameWithSlash(root, truncated)
	return name, flags
}

// nameWithSlash converts the first underscore in a model name to a slash.
// If there is a dash, only top domain names between the dash and the slash.
//
//nolint:revive // will refactor/split to reduce cognitive complexity (31).
func nameWithSlash(root, truncated string) string {
	name := filepath.Base(truncated)

	pos := -1

	for i, char := range name {
		if i > 9 {
			if pos < 0 { // the limit is 9 letters without a dash
				return nameWithDir(root, truncated, name)
			}
			if i > 11 { // otherwise the limit is 10 letters + one dash
				return nameWithDir(root, truncated, name)
			}
		}

		switch {
		case unicode.IsLower(char):
			continue
		case char == '-': // dash
			if i < 4 {
				return nameWithDir(root, truncated, name)
			}
			if pos > -1 {
				return nameWithDir(root, truncated, name)
			}
			pos = i
		case char == '_': // underscore
			if pos > 0 {
				n := i - pos // number of letters before the dash
				ok := n == 3 || n == 4
				if !ok {
					return nameWithDir(root, truncated, name)
				}
			}
			if i < 4 {
				return nameWithDir(root, truncated, name)
			}
			if i-pos < 3 {
				return nameWithDir(root, truncated, name)
			}
			n := []byte(name)
			n[i] = '/'
			return string(n)
		default:
			return nameWithDir(root, truncated, name)
		}
	}
	return nameWithDir(root, truncated, name)
}

// nameWithDir prefixes the model name with its folder name.
// If there is a dash, only top domain names between the dash and the slash.
func nameWithDir(root, truncated, name string) string {
	dir := filepath.Dir(truncated)
	if len(dir) <= len(root) {
		return name
	}
	dir = filepath.Base(dir)
	pos := -1
	for i, char := range dir {
		switch {
		case i > 12:
			return name
		case unicode.IsLower(char):
			continue
		case char == '-':
			if i < 4 {
				return name
			}
			if pos > -1 {
				return name
			}
			pos = i
		default:
			return name
		}
	}
	if pos > 0 {
		n := len(dir) - pos // number of letters before the dash
		ok := n == 3 || n == 4
		if !ok {
			return name
		}
	}
	return dir + "/" + name
}

// extractFlags returns the truncated path and the llama-server flags from a file path.
// It first checks for a companion ".args" file; if present, its contents are used as flags.
// Otherwise, it parses flags encoded in the filename after an '&' delimiter.
// Returns the truncated path (without extension) and a space‑separated flag string.
//
//nolint:gocritic,revive // return the truncated model filename and the llama-server flags.
func extractFlags(path string) (string, string) {
	truncated := strings.TrimSuffix(path, ".gguf")

	// Huge GGUF are spilt into smaller files ending with -00001-of-00003.gguf
	pos := strings.LastIndex(truncated, "-00001-of-")
	if pos > 0 {
		truncated = truncated[:pos]
	}

	// 1. Is there a file containing the command line arguments?
	args, err := os.ReadFile(filepath.Clean(truncated + ".args"))
	if err == nil {
		return truncated, oneLine(args)
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
		kv := strings.SplitN(f, "=", 2)
		if len(kv) > 0 {
			kv[0] = "-" + kv[0]
			flags = append(flags, kv...)
		}
	}

	return truncated[:pos], strings.Join(flags, " ")
}

// oneLine converts the `.args` file into a single space‑separated string,
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
		// Skip blank lines and comments
		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}
		// Convert the byte slice to a string before appending.
		keep = append(keep, line...)
		keep = append(keep, ' ')
	}

	return string(keep)
}

// countModels returns the number of models that are currently present on file system.
func (cfg *Cfg) countModels() int {
	modelFiles, err := cfg.search()
	if err != nil {
		return 0
	}
	return len(modelFiles)
}

// ValidateSwap checks that the configuration contains at least one model file and
// that each model referenced in the swap configuration exists on disk.
// It logs warnings and errors as appropriate.
func (cfg *Cfg) ValidateSwap() error {
	if len(cfg.Swap.Models) == 0 {
		n := cfg.countModels()
		if n == 0 {
			slog.Error("No *.gguf files found", "dir", cfg.ModelsDir)
			return gie.ErrConfigValidation
		}
		slog.Warn("No model configured => Use flag -gen-swap-cfg to fill the config with", "models", n)
		return nil
	}

	for i := range cfg.Swap.Models {
		var previous string
		for arg := range strings.SplitSeq(cfg.Swap.Models[i].Cmd, " ") {
			if previous == "-m" || previous == "--model" {
				err := validateFile(arg)
				if err != nil {
					return err
				}
			}
			previous = arg
		}
	}
	return nil
}

// validateFile verifies that the given path points to
// an existing, readable, and sufficiently large *.gguf file.
// It also normalizes the path and checks for series files.
func validateFile(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned != path {
		slog.Warn("Malformed", "current", path, "better", cleaned)
		path = cleaned
	}

	// Check if the file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		slog.Warn("Model file does not exist", "path", path)
		return err
	}

	// Check if the file is readable
	file, err := os.Open(path)
	if err != nil {
		slog.Warn("Model file is not readable", "path", path)
		return err
	}

	err = file.Close()
	if err != nil {
		slog.Warn("Model file fails closing", "path", path)
		return err
	}

	// is empty?
	if info.Size() < 1000 {
		slog.Warn("Model file is empty (or too small)", "path", path)
		return gie.ErrConfigValidation
	}

	// Huge GGUF are spilt into smaller files ending with -00001-of-00003.gguf
	// Keep only the first one, and ignore the others: -00002-of-00003.gguf
	pos := strings.LastIndex(path, "-of-")
	const first = "00001"
	if pos < len(first) {
		return nil // OK
	}

	if path[pos-len(first):pos] != first {
		slog.Debug("KO Model file is part of a series, but only the first one is referenced", "path", path)
		return gie.New(gie.ConfigErr, "Model file is part of a series, but only the first one is referenced, file="+path)
	}

	slog.Debug("OK Model file is the first of a series", "path", path)
	return nil // OK
}
