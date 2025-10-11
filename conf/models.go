// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package conf

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/LM4eu/goinfer/gie"
)

type (
	// ModelInfo is used for the response of the /models endpoint, including:
	// - command‑line flags found of file system
	// - eventual error (if the model is missing or misconfigured).
	ModelInfo struct {
		Template *TemplateInfo `json:"template,omitempty" yaml:"template,omitempty"`
		Flags    string        `json:"cmd,omitempty"      yaml:"cmd,omitempty"`
		Path     string        `json:"path,omitempty"     yaml:"path,omitempty"`
		Error    string        `json:"error,omitempty"    yaml:"error,omitempty"`
		Size     int64         `json:"size,omitempty"     yaml:"size,omitempty"`
	}

	TemplateInfo struct {
		Name  string `json:"name,omitempty"  yaml:"name,omitempty"`
		Flags string `json:"flags,omitempty" yaml:"flags,omitempty"`
		Error string `json:"error,omitempty" yaml:"error,omitempty"`
		Ctx   int    `json:"ctx,omitempty"   yaml:"ctx,omitempty"`
	}
)

const notConfigured = "file present but not configured in llama-swap.yml"

// ListModels returns the model names from the config and from the models_dir.
func (cfg *Cfg) ListModels() map[string]ModelInfo {
	if len(cfg.Info) > 0 {
		return cfg.Info
	}

	cfg.updateInfo()
	for name, mi := range cfg.Info {
		if cfg.Info[name].Error == "" {
			mi.Error = notConfigured
			cfg.Info[name] = mi
		}
	}

	for name := range cfg.Swap.Models {
		if len(name) > 3 && name[:3] == "GI_" && cfg.Swap.Models[name].Unlisted {
			continue // do not report models for /completion endpoint
		}
		cfg.refineModelInfo(name)
	}

	return cfg.Info
}

func (cfg *Cfg) refineModelInfo(name string) {
	mi, ok := cfg.Info[name]
	if ok {
		if mi.Error == notConfigured {
			mi.Error = "" // OK: model is both present in FS and configured in llama-swap.yml
		}
		cfg.Info[name] = mi
		return
	}

	if mi.Flags != "" {
		return // change nothing
	}

	// after the first space, the arguments
	pos := strings.Index(cfg.Swap.Models[name].Cmd, " ")
	if pos > 1 {
		// split the arguments at -m: -first -args -m path/to/file.gguf
		args := strings.SplitN(cfg.Swap.Models[name].Cmd[pos:], " -m ", 2)
		mi.Flags = args[0]
		if len(args) > 1 {
			mi.Path = args[1]
			mi.Error = "file absent but configured in llama-swap.yml"
		}
	} else {
		slog.Warn("missing space characters", "cmd", cfg.Swap.Models[name].Cmd)
		mi.Error = "missing space characters in cmd=" + cfg.Swap.Models[name].Cmd
	}
	cfg.Info[name] = mi
}

// updateInfo search template.yml and *.gguf model files recursively
// in the directories listed in cfg.ModelsDir (colon-separated).
// It aggregates matching files, and updates cfg.Info.
func (cfg *Cfg) updateInfo() {
	templates := map[string]TemplateInfo{}
	if cfg.Info == nil {
		cfg.Info = make(map[string]ModelInfo, 16)
	} else {
		clear(cfg.Info)
	}

	// collect templates.yml and GUFF files
	for root := range strings.SplitSeq(cfg.ModelsDir, ":") {
		err := cfg.search(templates, strings.TrimSpace(root))
		if err != nil {
			slog.Warn("cannot search files in", "root", root, "err", err)
		}
	}

	// Put the TemplateInfo in the corresponding ModelInfo
	for name, ti := range templates {
		mi := cfg.Info[name]
		mi.Template = &ti
		if mi.Flags != "" {
			mi.Flags = ti.Flags
			ti.Flags = ""
		}
		cfg.Info[name] = mi
	}
}

// search walks the given root directory and appends any valid *.gguf model file to
// cfg.Info. It validates each file using validateFile and warns about errors (logs).
func (cfg *Cfg) search(templates map[string]TemplateInfo, root string) error {
	return filepath.WalkDir(root, func(path string, dir fs.DirEntry, err error) error {
		switch {
		case err != nil:
			if dir == nil {
				return gie.Wrap(err, gie.NotFound, "filepath.WalkDir")
			}
			return gie.Wrap(err, gie.NotFound, "filepath.WalkDir", "dir", dir.Name())
		case dir.IsDir():
			// => step into this directory
		case filepath.Base(path) == "templates.yml":
			err = keepTemplates(templates, root, path)
			if err != nil {
				slog.Warn("skip template file", "path", path, "err", err)
			}
		case strings.HasSuffix(path, ".gguf"):
			cfg.keepGUFF(root, path)
		default:
		}
		return nil
	})
}

func keepTemplates(templates map[string]TemplateInfo, root, path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "os.ReadFile", "file", path)
	}

	if len(data) == 0 {
		slog.Info("Empty template", "file", path)
		return nil
	}

	slog.Debug("Found", "template", path)

	var tpl map[string]TemplateInfo
	err = json.Unmarshal(data, &tpl)
	if err != nil {
		return gie.Wrap(err, gie.ConfigErr, "json.Unmarshal", "file", path, "100FirsBytes", string(data[:100]))
	}

	for name, ti := range tpl {
		if old, ok := templates[name]; ok {
			slog.Warn("Duplicated templates", "dir", root, "name", name, "old", old, "new", ti)
			ti.Error = "two files have same model name (must be unique)"
		}
		ti.Flags = replaceDIR(path, ti.Flags)
		templates[name] = ti
	}

	return nil
}

func (cfg *Cfg) keepGUFF(root, path string) {
	size, err := verify(path)
	if err != nil {
		slog.Debug("skip GGUF", "path", path, "err", err)
		return
	}

	slog.Debug("Found", "model", path)

	name, flags := getNameAndFlags(root, path)

	flags = replaceDIR(path, flags)

	mi := ModelInfo{Template: nil, Flags: flags, Path: path, Error: "", Size: size}
	if old, ok := cfg.Info[name]; ok {
		slog.Warn("Duplicated models", "dir", root, "name", name, "old", old, "new", mi)
		mi.Error = "two files have same model name (must be unique)"
	}
	cfg.Info[name] = mi
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
	if len(cfg.Info) == 0 {
		cfg.updateInfo()
	}
	return len(cfg.Info)
}

// ValidateSwap checks that the configuration contains at least one model file and
// that each model referenced in the swap configuration exists on disk.
// It logs warnings and errors as appropriate.
func (cfg *Cfg) ValidateSwap() error {
	if len(cfg.Swap.Models) == 0 {
		n := cfg.countModels()
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
		return 0, gie.New(gie.ConfigErr, "Model file is empty (or too small)", "path", path)
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
