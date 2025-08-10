package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLlamaConfig_Validation(t *testing.T) {
	testCases := []struct {
		name   string
		config LlamaConf
		valid  bool
		errMsg string
	}{
		{
			name: "Valid minimal config",
			config: LlamaConf{
				BinaryPath: "./llama-server",
				ModelPath:  "./model.bin",
				Host:       "localhost",
				Port:       8080,
			},
			valid: true,
		},
		{
			name: "Valid config with args",
			config: LlamaConf{
				BinaryPath: "./llama-server",
				ModelPath:  "./model.bin",
				Host:       "localhost",
				Port:       8080,
				Args:       []string{"--ctx-size", "2048"},
			},
			valid: true,
		},
		{
			name: "Empty binary path",
			config: LlamaConf{
				BinaryPath: "",
				ModelPath:  "./model.bin",
				Host:       "localhost",
				Port:       8080,
			},
			valid:  false,
			errMsg: "binary path cannot be empty",
		},
		{
			name: "Empty model path",
			config: LlamaConf{
				BinaryPath: "./llama-server",
				ModelPath:  "",
				Host:       "localhost",
				Port:       8080,
			},
			valid:  false,
			errMsg: "model path cannot be empty",
		},
		{
			name: "Empty host",
			config: LlamaConf{
				BinaryPath: "./llama-server",
				ModelPath:  "./model.bin",
				Host:       "",
				Port:       8080,
			},
			valid:  false,
			errMsg: "host cannot be empty",
		},
		{
			name: "Invalid port format",
			config: LlamaConf{
				BinaryPath: "./llama-server",
				ModelPath:  "./model.bin",
				Host:       "localhost",
				Port:       -123, // invalid
			},
			valid:  false,
			errMsg: "port must be a valid number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestLlamaConfig_Clone(t *testing.T) {
	original := LlamaConf{
		BinaryPath: "./llama-server",
		ModelPath:  "./model.bin",
		Host:       "localhost",
		Port:       8080,
		Args:       []string{"--ctx-size", "2048"},
	}

	// Clone the config
	cloned := original.Clone()

	// Verify they are equal (compare pointers)
	assert.Equal(t, &original, cloned)

	// Modify the clone
	cloned.Args = append(cloned.Args, "--threads", "4")

	// Verify they are now different
	assert.NotEqual(t, original.Args, cloned.Args)
	assert.Equal(t, []string{"--ctx-size", "2048"}, original.Args)
	assert.Equal(t, []string{"--ctx-size", "2048", "--threads", "4"}, cloned.Args)
}


