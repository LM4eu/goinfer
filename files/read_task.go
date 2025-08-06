package files

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// keyExists checks if a key exists in a map
func keyExists(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

// convertTask converts a map to a Task type with proper error handling
func convertTask(m map[string]interface{}) (types.Task, error) {
	// Validate required fields
	name, ok := m["name"].(string)
	if !ok {
		return types.Task{}, fmt.Errorf("task name is required and must be a string")
	}

	template, ok := m["template"].(string)
	if !ok {
		return types.Task{}, fmt.Errorf("task template is required and must be a string")
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
							}
						case "ctx":
							if ctx, ok := v.(int); ok {
								task.ModelConf.Ctx = int(ctx)
							}
						case "gpu_layers":
							if gpuLayers, ok := v.(int); ok {
								task.ModelConf.GPULayers = int(gpuLayers)
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
							}
						case "threads":
							if threads, ok := v.(int); ok {
								ip.Threads = int(threads)
							}
						case "n_predict":
							if npredict, ok := v.(int); ok {
								ip.NPredict = int(npredict)
							}
						case "top_k":
							if topk, ok := v.(int); ok {
								ip.TopK = int(topk)
							}
						case "top_p":
							if topp, ok := v.(float64); ok {
								ip.TopP = float32(topp)
							}
						case "temperature":
							if temp, ok := v.(float64); ok {
								ip.Temperature = float32(temp)
							}
						case "frequency_penalty":
							if freqPenalty, ok := v.(float64); ok {
								ip.FrequencyPenalty = float32(freqPenalty)
							}
						case "presence_penalty":
							if presPenalty, ok := v.(float64); ok {
								ip.PresencePenalty = float32(presPenalty)
							}
						case "repeat_penalty":
							if repeatPenalty, ok := v.(float64); ok {
								ip.RepeatPenalty = float32(repeatPenalty)
							}
						case "tfs_z":
							if tfs, ok := v.(float64); ok {
								ip.TailFreeSamplingZ = float32(tfs)
							}
						case "stop":
							if stopSlice, ok := v.([]interface{}); ok {
								stop := make([]string, len(stopSlice))
								for i, val := range stopSlice {
									stop[i] = fmt.Sprint(val)
								}
								ip.StopPrompts = stop
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

// ReadTask reads a task from a YAML file with proper error handling
func ReadTask(path string) (bool, types.Task, error) {
	m := make(map[string]interface{})
	//p := filepath.Join(state.TasksDir, path) //TODO
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
