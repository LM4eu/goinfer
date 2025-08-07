package files

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
	"gopkg.in/yaml.v3"
)

// keyExists checks if a key exists in a map.
func keyExists(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

// convertToInt converts an interface{} to int with support for float64 conversion.
func convertToInt(value interface{}, paramName string, context string) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		// Check if the float64 has no fractional part (can be safely converted to int)
		if v == float64(int(v)) {
			return int(v), nil
		}
		return 0, fmt.Errorf("%s %s must be an integer or a float64 without fractional part, got %f", context, paramName, v)
	default:
		return 0, fmt.Errorf("%s %s must be an integer or float64, got %T", context, paramName, value)
	}
}

// convertTask converts a map to a Task type with proper error handling.
func convertTask(m map[string]interface{}) (types.Task, error) {
	// Validate required fields
	name, ok := m["name"].(string)
	if !ok {
		return types.Task{}, errors.New("task name is required and must be a string")
	}

	template, ok := m["template"].(string)
	if !ok {
		return types.Task{}, errors.New("task template is required and must be a string")
	}

	task := types.Task{
		Name:        name,
		ModelConf:   state.DefaultModelConf,
		Template:    template,
		InferParams: state.DefaultInferenceParams,
	}

	// Process modelConf if present
	if keyExists(m, "modelConf") {
		if modelConfRaw, ok := m["modelConf"].([]interface{}); ok {
			for _, param := range modelConfRaw {
				if paramMap, ok := param.(map[string]interface{}); ok {
					for k, v := range paramMap {
						switch k {
						case "name":
							if name, ok := v.(string); ok {
								task.ModelConf.Name = name
							} else {
								return types.Task{}, fmt.Errorf("modelConf name must be a string, got %T", v)
							}
						case "ctx":
							if ctx, err := convertToInt(v, "ctx", "modelConf"); err != nil {
								return types.Task{}, err
							} else {
								task.ModelConf.Ctx = ctx
							}
						case "gpu_layers":
							if gpuLayers, err := convertToInt(v, "gpu_layers", "modelConf"); err != nil {
								return types.Task{}, err
							} else {
								task.ModelConf.GPULayers = gpuLayers
							}
						}
					}
				}
			}
		}
	}

	// Process inferParams if present
	ip := state.DefaultInferenceParams
	if keyExists(m, "inferParams") {
		if inferParamsRaw, ok := m["inferParams"].([]interface{}); ok {
			for _, param := range inferParamsRaw {
				if paramData, ok := param.(map[string]interface{}); ok {
					for k, v := range paramData {
						switch k {
						case "stream":
							if stream, ok := v.(bool); ok {
								ip.Stream = stream
							} else {
								return types.Task{}, fmt.Errorf("inferParams stream must be a boolean, got %T", v)
							}
						case "threads":
							if threads, err := convertToInt(v, "threads", "inferParams"); err != nil {
								return types.Task{}, err
							} else {
								ip.Threads = threads
							}
						case "n_predict":
							if npredict, err := convertToInt(v, "n_predict", "inferParams"); err != nil {
								return types.Task{}, err
							} else {
								ip.NPredict = npredict
							}
						case "top_k":
							if topk, err := convertToInt(v, "top_k", "inferParams"); err != nil {
								return types.Task{}, err
							} else {
								ip.TopK = topk
							}
						case "top_p":
							if topp, ok := v.(float64); ok {
								ip.TopP = float32(topp)
							} else {
								return types.Task{}, fmt.Errorf("inferParams top_p must be a float64, got %T", v)
							}
						case "temperature":
							if temp, ok := v.(float64); ok {
								ip.Temperature = float32(temp)
							} else {
								return types.Task{}, fmt.Errorf("inferParams temperature must be a float64, got %T", v)
							}
						case "frequency_penalty":
							if freqPenalty, ok := v.(float64); ok {
								ip.FrequencyPenalty = float32(freqPenalty)
							} else {
								return types.Task{}, fmt.Errorf("inferParams frequency_penalty must be a float64, got %T", v)
							}
						case "presence_penalty":
							if presPenalty, ok := v.(float64); ok {
								ip.PresencePenalty = float32(presPenalty)
							} else {
								return types.Task{}, fmt.Errorf("inferParams presence_penalty must be a float64, got %T", v)
							}
						case "repeat_penalty":
							if repeatPenalty, ok := v.(float64); ok {
								ip.RepeatPenalty = float32(repeatPenalty)
							} else {
								return types.Task{}, fmt.Errorf("inferParams repeat_penalty must be a float64, got %T", v)
							}
						case "tfs_z":
							if tfs, ok := v.(float64); ok {
								ip.TailFreeSamplingZ = float32(tfs)
							} else {
								return types.Task{}, fmt.Errorf("inferParams tfs_z must be a float64, got %T", v)
							}
						case "stop":
							if stopSlice, ok := v.([]interface{}); ok {
								stop := make([]string, len(stopSlice))
								for i, val := range stopSlice {
									stop[i] = fmt.Sprint(val)
								}
								ip.StopPrompts = stop
							} else {
								return types.Task{}, fmt.Errorf("inferParams stop must be a slice, got %T", v)
							}
						}
					}
				}
			}
		}
	}

	task.InferParams = ip

	return task, nil
}

// ReadTask reads a task from a YAML file with proper error handling.
func ReadTask(path string) (bool, types.Task, error) {
	m := make(map[string]interface{})
	// p := filepath.Join(state.TasksDir, path) //TODO
	p := state.TasksDir + "/" + path
	_, err := os.Stat(p)
	var t types.Task

	if os.IsNotExist(err) {
		return false, t, nil
	}

	file, err := os.Open(p)
	if err != nil {
		return false, t, fmt.Errorf("failed to open task file %s: %w", p, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return false, t, fmt.Errorf("failed to read task file %s: %w", p, err)
	}

	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return false, t, fmt.Errorf("failed to unmarshal task file %s: %w", p, err)
	}

	t, err = convertTask(m)
	if err != nil {
		return false, t, fmt.Errorf("failed to convert task from file %s: %w", p, err)
	}

	return true, t, nil
}
