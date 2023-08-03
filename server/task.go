package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-skynet/go-llama.cpp"
	"github.com/labstack/echo/v4"
	"github.com/synw/altiplano/goinfer/files"
	"github.com/synw/altiplano/goinfer/lm"
	"github.com/synw/altiplano/goinfer/state"
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
	res, err := lm.Infer(prompt, task.Template, task.InferParams)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, res)
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
