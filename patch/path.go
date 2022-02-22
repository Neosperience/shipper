package patch

import (
	"errors"
	"strings"
)

func SetPath(root map[string]interface{}, path string, value interface{}) error {
	pieces := strings.Split(path, ".")
	data := root
	head, tail := pieces[:len(pieces)-1], pieces[len(pieces)-1]

	for _, piece := range head {
		_, ok := data[piece]
		if !ok {
			data[piece] = make(map[string]interface{})
		}
		switch v := data[piece].(type) {
		case map[string]interface{}:
			data = v
		default:
			return errors.New("found a value while traversing a tree")
		}
	}

	data[tail] = value
	return nil
}
