package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Node struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Path     string  `json:"path"`
	Children []*Node `json:"children,omitempty"`
}

var keyCounter int

func NewNode(label string, path string) *Node {
	keyCounter++
	return &Node{
		Key:   fmt.Sprintf("%d", keyCounter),
		Label: strings.Replace(label, ".yml", "", 1),
		Path:  path,
	}
}

func addPath(root *Node, path string) {
	parts := strings.Split(path, string(filepath.Separator))
	current := root
	prevParts := ""
	for _, part := range parts {
		found := false
		for _, child := range current.Children {
			if child.Label == part {
				current = child
				found = true
				break
			}
		}
		if part != root.Label {
			if len(prevParts) > 0 {
				prevParts = prevParts + "/" + part
			} else {
				prevParts = part
			}
		}

		if !found {
			newNode := NewNode(part, prevParts)
			current.Children = append(current.Children, newNode)
			current = newNode
		}
	}
}

func ReadTasks(rootPath string) ([]*Node, error) {
	keyCounter = 0

	// Clean the root path to ensure consistent handling
	rootPath = filepath.Clean(rootPath)
	root := NewNode(filepath.Base(rootPath), "")
	relRootPath := strings.Replace(rootPath, "./", "", 1)

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".yml" {
			// Get relative path from the root path
			relPath, err := filepath.Rel(relRootPath, path)
			if err != nil {
				return err
			}
			addPath(root, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	ts := []*Node{root}[0]
	data := ts.Children
	if len(ts.Children) > 0 {
		if len(ts.Children[0].Children) > 0 {
			data = ts.Children[0].Children
		}
	}
	return data, nil
}
