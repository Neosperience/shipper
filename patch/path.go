package patch

import (
	"errors"
	"strings"
)

var (
	ErrInvalidYAMLStructure = errors.New("found a value while traversing a tree")
)

func SetPath(root map[string]any, path string, value any) error {
	pieces := strings.Split(path, ".")
	data := root
	head, tail := pieces[:len(pieces)-1], pieces[len(pieces)-1]

	for _, piece := range head {
		_, ok := data[piece]
		if !ok {
			data[piece] = make(map[string]any)
		}
		switch v := data[piece].(type) {
		case map[string]any:
			data = v
		default:
			return ErrInvalidYAMLStructure
		}
	}

	data[tail] = value
	return nil
}
