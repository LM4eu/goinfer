package llama

import (
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// LlamaConfig - configuration for llama-server proxy.
type LlamaConfig struct {
	BinaryPath  string // Path to llama-server binary
	ModelPath   string // Path to model file
	ContextSize int
	GpuLayers   int
	DownloadUrl string   // model from HuggingFace
	Host        string   // Host binding (default: localhost)
	Port        int      // Port number (default: 8080)
	Args        []string // Additional arguments
}

// NewLlamaConfig - Creates a new LlamaConfig with minimal validation.
func NewLlamaConfig(binaryPath, modelPath string, args ...string) *LlamaConfig {
	return &LlamaConfig{
		BinaryPath: binaryPath,
		ModelPath:  modelPath,
		Host:       "localhost",
		Port:       8080,
		Args:       args,
	}
}

// GetAddress - Returns the server address in host:port format.
func (c *LlamaConfig) GetAddress() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// GetCommandArgs - Returns the complete command arguments for llama-server.
func (c *LlamaConfig) GetCommandArgs() []string {
	args := make([]string, 0, len(c.Args)+5)
	args = append(args, "-m", c.ModelPath)
	args = append(args, "-h", c.Host)
	args = append(args, "-p", strconv.Itoa(c.Port))

	args = append(args, c.Args...)
	return args
}

func (c *LlamaConfig) Validate() error {
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

// ErrInvalidConfig - Error type for configuration validation.
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + string(e)
}

// Clone - cloning for configuration updates.
func (c *LlamaConfig) Clone() *LlamaConfig {
	// Pre-allocate slice to avoid allocations
	args := make([]string, len(c.Args))
	copy(args, c.Args)

	return &LlamaConfig{
		BinaryPath: c.BinaryPath,
		ModelPath:  c.ModelPath,
		Host:       c.Host,
		Port:       c.Port,
		Args:       args,
	}
}

// MergeArgs - Efficiently merge additional arguments.
func (c *LlamaConfig) MergeArgs(additional []string) {
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
func (c *LlamaConfig) HasArg(arg string) bool {
	for _, existing := range c.Args {
		if existing == arg {
			return true
		}
	}
	return false
}

// GetArgValue - retrieval of argument value (key=value format).
func (c *LlamaConfig) GetArgValue(key string) string {
	prefix := key + "="
	for _, arg := range c.Args {
		if strings.HasPrefix(arg, prefix) {
			return arg[len(prefix):]
		}
	}
	return ""
}

// GetCommand - Returns a command for llama-server execution.
func (c *LlamaConfig) GetCommand() *exec.Cmd {
	return exec.Command(c.BinaryPath, c.GetCommandArgs()...)
}
