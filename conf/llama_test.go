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
				ExePath:       "./llama-server",
				ModelPathname: "./model.bin",
			},
			valid: true,
		},
		{
			name: "Valid config with args",
			config: LlamaConf{
				ExePath:       "./llama-server",
				ModelPathname: "./model.bin",
				Args:          []string{"--ctx-size", "2048"},
			},
			valid: true,
		},
		{
			name: "Empty binary path",
			config: LlamaConf{
				ExePath:       "",
				ModelPathname: "./model.bin",
			},
			valid:  false,
			errMsg: "binary path cannot be empty",
		},
		{
			name: "Empty model path",
			config: LlamaConf{
				ExePath:       "./llama-server",
				ModelPathname: "",
			},
			valid:  false,
			errMsg: "model path cannot be empty",
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
