package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

func ExecuteTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	var path string
	v, ok := m["task"]
	if ok {
		path = v.(string)
		if !strings.HasSuffix(path, ".yml") {
			path = path + ".yml"
		}
	}
	var prompt string
	v, ok = m["prompt"]
	if ok {
		prompt = v.(string)
	}
	var instruction = ""
	v, ok = m["instruction"]
	if ok {
		instruction = "\n\n" + v.(string)
	}
	exists, task, err := files.ReadTask(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	if !exists {
		return c.JSON(http.StatusBadRequest, err)
	}
	task.Template = strings.Replace(task.Template, "{instruction}", instruction, 1)
	if state.IsInfering {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}
	// check if the model is loaded
	loadModel := true
	if state.IsModelLoaded {
		if state.LoadedModel == task.Model {
			if state.ModelConf.Ctx == task.ModelConf.Ctx {
				loadModel = false
			}
		}
	}
	if loadModel {
		lm.LoadModel(task.Model, llama.ModelOptions{
			ContextSize: task.ModelConf.Ctx,
		})
	}
	// exec task
	ch := make(chan types.StreamedMessage)
	errCh := make(chan types.StreamedMessage)
	defer close(ch)
	defer close(errCh)

	go lm.Infer(prompt, task.Template, task.InferParams, c, ch, errCh)

	select {
	case res, ok := <-ch:
		if ok {
			if state.IsVerbose {
				fmt.Println("-------- result ----------")
				for key, value := range res.Data {
					fmt.Printf("%s: %v\n", key, value)
				}
				fmt.Println("--------------------------")
			}
			if !task.InferParams.Stream {
				return c.JSON(http.StatusOK, res.Data)
			}
		}
		return nil
	case err, ok := <-errCh:
		if ok {
			return c.JSON(http.StatusInternalServerError, err)
		}
		return nil
	case <-c.Request().Context().Done():
		fmt.Println("\nRequest canceled")
		state.ContinueInferingController = false
		return c.NoContent(http.StatusNoContent)
	}
}

func ReadTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	var path string
	v, ok := m["path"]
	if ok {
		path = v.(string)
	}
	exists, task, err := files.ReadTask(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	if !exists {
		return c.JSON(http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, task)
}

func ReadTasksHandler(c echo.Context) error {
	tasks, err := files.ReadTasks(state.TasksDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, tasks)
}

func SaveTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return err
	}
	var name string
	v, ok := m["name"]
	if ok {
		name = v.(string)
	}
	var model string
	v, ok = m["model"]
	if ok {
		model = v.(string)
	}
	var template string
	v, ok = m["template"]
	if ok {
		template = v.(string)
	}
	var ctx float32
	v, ok = m["ctx"]
	if ok {
		ctx = float32(v.(float64))
	}
	modelConf := types.ModelConf{
		Ctx: int(ctx),
	}
	var rawInferParams map[string]interface{}
	v, ok = m["inferParams"]
	if ok {
		rawInferParams = v.(map[string]interface{})
		rawInferParams["template"] = template
	}
	rawInferParams["prompt"] = ""
	_, _, inferParams, err := ParseInferParams(rawInferParams)
	if err != nil {
		return err
	}
	task := types.Task{
		Name:        name,
		Model:       model,
		Template:    template,
		ModelConf:   modelConf,
		InferParams: inferParams,
	}
	files.SaveTask(task)
	return c.NoContent(http.StatusCreated)
}
