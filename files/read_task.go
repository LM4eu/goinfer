package files

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func keyExists(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

func convertTask(m map[string]interface{}) (types.Task, error) {
	task := types.Task{
		Name:        m["name"].(string),
		ModelConf:   state.DefaultModelConf,
		Template:    m["template"].(string),
		InferParams: state.DefaultInferenceParams,
	}
	if keyExists(m, "modelConf") {
		rmc := m["modelConf"].([]interface{})
		for _, param := range rmc {
			mp := param.(map[string]interface{})
			for k, v := range mp {
				if k == "name" {
					task.ModelConf.Name = v.(string)
				} else if k == "ctx" {
					task.ModelConf.Ctx = v.(int)
				}
			}
		}
		ip := state.DefaultInferenceParams
		if keyExists(m, "inferParams") {
			rip := m["inferParams"].([]interface{})
			for _, param := range rip {
				ipr := param.(map[string]interface{})
				for k, v := range ipr {
					//fmt.Println("P", k, v)
					switch k {
					case "threads":
						ip.Threads = v.(int)
					case "n_predict":
						ip.NPredict = v.(int)
					case "top_k":
						ip.TopK = v.(int)
					case "top_p":
						ip.TopP = float32(v.(float64))
					case "temperature":
						ip.Temperature = float32(v.(float64))
					case "frequency_penalty":
						ip.FrequencyPenalty = float32(v.(float64))
					case "presence_penalty":
						ip.PresencePenalty = float32(v.(float64))
					case "repeat_penalty":
						ip.RepeatPenalty = float32(v.(float64))
					case "tfs_z":
						ip.TailFreeSamplingZ = float32(v.(float64))
					case "stop":
						s := v.([]interface{})
						t := []string{}
						for _, v = range s {
							t = append(t, v.(string))
						}
						ip.StopPrompts = t
					}
				}
			}
		}
		task.InferParams = ip
	}
	return task, nil
}

func ReadTask(path string) (bool, types.Task, error) {
	//var task types.Task
	m := make(map[string]interface{})
	//p := filepath.Join(state.TasksDir, path)
	p := state.TasksDir + "/" + path
	//fmt.Println("Opening", p)
	_, err := os.Stat(p)
	var t types.Task
	if os.IsNotExist(err) {
		return false, t, nil
	}
	file, err := os.Open(p)
	if err != nil {
		return true, t, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return true, t, err
	}
	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return true, t, err
	}

	t, err = convertTask(m)
	if err != nil {
		return true, t, err
	}
	// fmt.Println("Task:", t)
	return true, t, nil
}
