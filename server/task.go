package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/files"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// ExecuteTaskHandler executes a saved task.
func ExecuteTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind task execution parameters: %w", err)
	}

	var path string
	v, ok := m["task"]
	if ok {
		if p, ok := v.(string); ok {
			path = p
			if !strings.HasSuffix(path, ".yml") {
				path = path + ".yml"
			}
		}
	}

	var prompt string
	v, ok = m["prompt"]
	if ok {
		if p, ok := v.(string); ok {
			prompt = p
		}
	}

	instruction := ""
	v, ok = m["instruction"]
	if ok {
		if i, ok := v.(string); ok {
			instruction = i
		}
	}

	exists, task, err := files.ReadTask(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	if !exists {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": fmt.Sprintf("task %s not found", path),
		})
	}

	task.Template = strings.Replace(task.Template, "{instruction}", instruction, 1)

	if state.IsInferring {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}

	// check if the model is loaded
	loadModel := true

	if state.IsModelLoaded {
		if state.LoadedModel == task.ModelConf.Name {
			if state.ModelOptions.ContextSize == task.ModelConf.Ctx {
				loadModel = false
			}
		}
	}

	if loadModel {
		err := setModelOptions(task.ModelConf)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": fmt.Sprintf("failed to set model options: %v", err),
			})
		}

		_, err = lm.LoadModel(task.ModelConf.Name, state.ModelOptions)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": fmt.Sprintf("failed to load model: %v", err),
			})
		}
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
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": err.Content,
			})
		}
		return nil
	case <-c.Request().Context().Done():
		fmt.Println("\nRequest canceled")
		state.ContinueInferringController = false
		return c.NoContent(http.StatusNoContent)
	}
}

// ReadTaskHandler reads a specific task.
func ReadTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind read task parameters: %w", err)
	}

	var path string
	v, ok := m["path"]
	if ok {
		if p, ok := v.(string); ok {
			path = p
			if !strings.HasSuffix(path, ".yml") {
				path = path + ".yml"
			}
		}
	}

	exists, task, err := files.ReadTask(path)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	if !exists {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": fmt.Sprintf("task %s not found", path),
		})
	}

	return c.JSON(http.StatusOK, task)
}

// ReadTasksHandler reads all available tasks.
func ReadTasksHandler(c echo.Context) error {
	tasks, err := files.ReadTasks(state.TasksDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, tasks)
}

// SaveTaskHandler saves a task.
func SaveTaskHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind save task parameters: %w", err)
	}

	var name string
	v, ok := m["name"]
	if ok {
		if n, ok := v.(string); ok {
			name = n
		}
	}

	var template string
	v, ok = m["template"]
	if ok {
		if t, ok := v.(string); ok {
			template = t
		}
	}

	var rawInferParams map[string]interface{}
	v, ok = m["inferParams"]
	if ok {
		if params, ok := v.(map[string]interface{}); ok {
			rawInferParams = params
			rawInferParams["template"] = template
		}
	}

	rawInferParams["prompt"] = ""

	_, _, modelConf, inferParams, err := ParseInferParams(rawInferParams)
	if err != nil {
		return fmt.Errorf("failed to parse inference parameters: %w", err)
	}

	task := types.Task{
		Name:        name,
		Template:    template,
		ModelConf:   modelConf,
		InferParams: inferParams,
	}

	err = files.SaveTask(task)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": fmt.Sprintf("failed to save task: %v", err),
		})
	}

	return c.NoContent(http.StatusCreated)
}
