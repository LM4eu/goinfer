package conf

import (
	"net"
	"strconv"
	"strings"
)

// LlamaConf - configuration for llama-server proxy.
type LlamaConf struct {
	BinaryPath     string // Path to llama-server binary
	Host           string // Host binding (default: localhost)
	Port           int    // Port number (default: 8080)
	ModelPath      string // Path to model file OR download url OR from HuggingFace repo
	ContextSize    int
	Threads        int  // number of threads to use during generation (default: -1)
	ThPromptProc   int  // number of threads to use during batch and prompt processing
	FlashAttention bool // enable Flash Attention
	GpuLayers      int
	WebUI          bool
	Args           []string // Additional arguments
}

// ErrInvalidConfig - Error type for configuration validation.
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + string(e)
}

// GetAddress - Returns the server address in host:port format.
func (c *LlamaConf) GetAddress() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// GetCommandArgs - Returns the complete command arguments for llama-server.
func (c *LlamaConf) GetCommandArgs() []string {
	args := make([]string, 0, len(c.Args)+5)
	args = append(args, "-h", c.Host)
	args = append(args, "-p", strconv.Itoa(c.Port))
	args = append(args, "--props") // enable changing global properties via POST /props

	if c.Threads != 0 {
		args = append(args, "-t", strconv.Itoa(c.Threads))
	}
	if c.ThPromptProc != 0 {
		args = append(args, "-tb", strconv.Itoa(c.ThPromptProc))
	}

	if c.ModelPath != "" {
		switch IsDownloadURL(c.ModelPath) {
		case 0:
			args = append(args, "-m", c.ModelPath)
		case 1:
			args = append(args, "-mu", c.ModelPath)
		default:
			args = append(args, "-hf", c.ModelPath)
		}
	}

	if c.ContextSize != 0 {
		args = append(args, "-c", strconv.Itoa(c.ContextSize))
	}
	if c.GpuLayers != 0 {
		args = append(args, "-ngl", strconv.Itoa(c.GpuLayers))
	}
	if !c.WebUI {
		args = append(args, "--no-webui")
	}
	if c.FlashAttention {
		args = append(args, "-fa")
	}

	args = append(args, c.Args...)
	return args
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
	// Only essential validation - paths and basic network checks
	if c.BinaryPath == "" {
		return ErrInvalidConfig("binary path cannot be empty")
	}
	if c.ModelPath == "" {
		return ErrInvalidConfig("model path cannot be empty")
	}
	if c.Host == "" {
		return ErrInvalidConfig("host cannot be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return ErrInvalidConfig("port must be a valid number")
	}

	// Fast network validation
	if c.Host != "" {
		if _, _, err := net.SplitHostPort(net.JoinHostPort(c.Host, strconv.Itoa(c.Port))); err != nil {
			return ErrInvalidConfig("invalid host:port combination")
		}
	}

	return nil
}

// Clone - cloning for configuration updates.
func (c *LlamaConf) Clone() *LlamaConf {
	// Pre-allocate slice to avoid allocations
	args := make([]string, len(c.Args))
	copy(args, c.Args)

	return &LlamaConf{
		BinaryPath: c.BinaryPath,
		ModelPath:  c.ModelPath,
		Host:       c.Host,
		Port:       c.Port,
		Args:       args,
	}
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
