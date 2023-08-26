package files

import (
	"fmt"
	"os"
	"sort"
)

func ReadModels(dir string) ([]string, error) {
	var modelFiles []string
	f, err := os.Open(dir)
	if err != nil {
		fmt.Println(err)
		return modelFiles, err
	}
	files, err := f.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return modelFiles, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size() < files[j].Size()
	})
	for _, v := range files {
		filename := v.Name()
		modelFiles = append(modelFiles, filename)
	}
	return modelFiles, nil
}
