package files

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/synw/goinfer/state"
	"github.com/synw/goinfer/types"
	"gopkg.in/yaml.v3"
)

func SaveTask(task types.Task) error {
	if state.IsVerbose {
		fmt.Println("Saving task", task)
	}
	filePath := state.TasksDir + "/" + task.Name
	dirPath := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	// Create directory if it doesn't exist
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(dirPath + "/" + fileName)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(file)
	err = enc.Encode(&task)
	if err != nil {
		return err
	}

	// Don't forget to close the file
	err = file.Close()
	if err != nil {
		return err
	}
	if state.IsVerbose {
		fmt.Println("Task", task.Name, "file successfully")
	}
	return nil
}
