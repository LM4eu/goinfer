package llama

import "github.com/synw/goinfer/types"

func (l *LlamaServerManager) Predict(query types.InferQuery, tokenCallback func(string) bool) (string, error) {
	return "", nil
}
