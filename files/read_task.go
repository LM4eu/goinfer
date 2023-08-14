package files

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func keyExists(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

func convertTask(m map[string]interface{}) (types.Task, error) {
	task := types.Task{
		Name:     m["name"].(string),
		Model:    m["model"].(string),
		Template: m["template"].(string),
	}
	if keyExists(m, "modelConf") {
		mc := types.ModelConf{}
		rmc := m["modelConf"].([]interface{})
		for _, param := range rmc {
			mp := param.(map[string]interface{})
			hasModelConf := false
			for k, v := range mp {
				if k == "ctx" {
					hasModelConf = true
					mc.Ctx = v.(int)
				}
			}
			if hasModelConf {
				task.ModelConf = mc
			}
		}
		ip := lm.DefaultInferenceParams
		if keyExists(m, "inferParams") {
			rip := m["inferParams"].([]interface{})
			for _, param := range rip {
				ipr := param.(map[string]interface{})
				for k, v := range ipr {
					fmt.Println("P", k, v)
					switch k {
					case "threads":
						ip.Threads = v.(int)
					case "tokens":
						ip.NPredict = v.(int)
					case "topK":
						ip.TopK = v.(int)
					case "topP":
						ip.TopP = float32(v.(float64))
					case "temp":
						ip.Temperature = float32(v.(float64))
					case "freqPenalty":
						ip.FrequencyPenalty = float32(v.(float64))
					case "presPenalty":
						ip.PresencePenalty = float32(v.(float64))
					case "tfs":
						ip.TailFreeSamplingZ = float32(v.(float64))
					case "stop":
						ip.StopPrompts = v.([]string)
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
