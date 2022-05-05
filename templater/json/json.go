package json_templater

import (
	"bytes"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
)

type FileUpdate struct {
	File string
	Path string
	Tag  string
}

type JSONProviderOptions struct {
	Ref     string
	Updates []FileUpdate
}

func UpdateJSONFile(repository targets.Repository, options JSONProviderOptions) (targets.FileList, error) {
	original := make(map[string][]byte)
	files := make(map[string]map[string]any)
	for _, update := range options.Updates {
		if _, ok := files[update.File]; !ok {
			file, err := repository.Get(update.File, options.Ref)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve %s from repository: %w", update.File, err)
			}

			original[update.File] = file
			data := make(map[string]any)
			if err := jsoniter.ConfigDefault.Unmarshal(file, &data); err != nil {
				return nil, fmt.Errorf("could not parse JSON file %s: %w", update.File, err)
			}
			files[update.File] = data
		}

		// Update the file
		files[update.File][update.Path] = update.Tag
	}

	diff := make(targets.FileList)
	for file, content := range files {
		byt, err := jsoniter.ConfigDefault.MarshalIndent(content, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("could not serialize modified file to JSON: %w", err)
		}

		// Skip if there are no changes
		if bytes.Equal(original[file], byt) {
			continue
		}

		diff[file] = byt
	}

	return diff, nil
}
