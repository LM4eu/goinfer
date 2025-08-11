package conf

import (
	"net"
	"strconv"
	"strings"
)

// LlamaConf - configuration for llama-server proxy.
type LlamaConf struct {
	ExePath       string   `json:"exe_path"       yaml:"exe_path"`       // Path to llama-server binary
	ModelPathname string   `json:"model_pathname" yaml:"model_pathname"` // Path to model file OR download url OR from HuggingFace repo
	PathnameType  int      `json:"pathname_type"  yaml:"pathname_type"`  // Positive=URL Zero=LocalFile Negative=HuggingFace repo
	ContextSize   int      `json:"ctx"            yaml:"ctx"`
	Threads       int      `json:"threads"        yaml:"threads"`       // number of threads to use during generation (default: -1)
	TPromptProc   int      `json:"t_prompt_proc"  yaml:"t_prompt_proc"` // use more threads to boost prompt processing
	Args          []string // Additional arguments
}

// ErrInvalidConfig - Error type for configuration validation.
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + string(e)
}

const (
	host = "127.0.0.1"
	port = "8080"
)

// GetAddress - Returns the server address in host:port format.
func (c *LlamaConf) GetAddress() string {
	return net.JoinHostPort(host, port)
}

// GetCommandArgs - Returns the complete command arguments for llama-server.
func (c *LlamaConf) GetCommandArgs() []string {
	args := make([]string, 0, len(c.Args)+14)
	args = append(args, "--host", host)
	args = append(args, "--port", port)
	args = append(args, "--props")    // enable changing global properties via POST /props
	args = append(args, "--no-webui") // no Web UI server

	if c.Threads != 0 {
		args = append(args, "-t", strconv.Itoa(c.Threads))
	}
	if c.TPromptProc != 0 {
		args = append(args, "-tb", strconv.Itoa(c.TPromptProc))
	}

	if c.ModelPathname != "" {
		if c.PathnameType > 0 {
			args = append(args, "-mu", c.ModelPathname)
		} else if c.PathnameType == 0 {
			args = append(args, "-m", c.ModelPathname)
		} else {
			args = append(args, "-hf", c.ModelPathname)
		}
		args = append(args, "-c", strconv.Itoa(c.ContextSize))
	}

	return append(args, c.Args...)
}

func IsDownloadURL(modelPath string) int {
	if modelPath[0:7] == "http://" || modelPath[0:7] == "https://" {
		return 1
	} else if modelPath[0] == '/' || modelPath[0:2] == "./" {
		return 0
	} else if strings.Contains(modelPath, "/") {
		return -1
	}
	return 0
}

func (c *LlamaConf) Validate() error {
	if c.ExePath == "" {
		return ErrInvalidConfig("binary path cannot be empty")
	}
	// TODO: is the model really required?
	if c.ModelPathname == "" {
		return ErrInvalidConfig("model path cannot be empty")
	}
	return nil
}

// MergeArgs - Efficiently merge additional arguments.
func (c *LlamaConf) MergeArgs(additional []string) {
	if len(additional) == 0 {
		return
	}

	// Pre-allocate new slice to avoid multiple allocations
	newArgs := make([]string, 0, len(c.Args)+len(additional))
	newArgs = append(newArgs, c.Args...)
	newArgs = append(newArgs, additional...)
	c.Args = newArgs
}

// HasArg - check for existing argument.
func (c *LlamaConf) HasArg(arg string) bool {
	for _, existing := range c.Args {
		if existing == arg {
			return true
		}
	}
	return false
}

// GetArgValue - retrieval of argument value (key=value format).
func (c *LlamaConf) GetArgValue(key string) string {
	prefix := key + "="
	for _, arg := range c.Args {
		if strings.HasPrefix(arg, prefix) {
			return arg[len(prefix):]
		}
	}
	return ""
}
