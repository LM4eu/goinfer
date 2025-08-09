package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/synw/goinfer/lm"
	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
)

// ExecutePromptHandler executes a saved task.
func ExecutePromptHandler(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		return fmt.Errorf("failed to bind task execution parameters: %w", err)
	}

	prompt, err := ParseInferParams(m)
	if err != nil {
		if state.IsDebug {
			fmt.Println("ParseInferParams error", err)
		}
		return c.NoContent(http.StatusBadRequest)
	}

	if state.IsInferring {
		fmt.Println("An inference query is already running")
		return c.NoContent(http.StatusAccepted)
	}

	// Check if the model is loaded
	loadModel := true

	if state.IsModelLoaded {
		if state.LoadedModel == prompt.ModelConf.Name {
			if state.ModelOptions.ContextSize == prompt.ModelConf.Ctx {
				loadModel = false
			}
		}
	}

	if loadModel {
		err := setModelOptions(prompt.ModelConf)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": fmt.Sprintf("failed to set model options: %v", err),
			})
		}

		_, err = lm.LoadModel(prompt.ModelConf.Name, state.ModelOptions)
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

	go lm.Infer(prompt, c, ch, errCh)

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
			if !prompt.InferParams.Stream {
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
