package kustomize_templater

import (
	"bytes"
	"fmt"

	"github.com/neosperience/shipper/targets"
	"gopkg.in/yaml.v3"
)

type KustomizeUpdate struct {
	KustomizationFile string
	Image             string
	NewImage          string
	NewTag            string
}

type KustomizeProviderOptions struct {
	Ref     string
	Updates []KustomizeUpdate
}

func UpdateKustomization(repository targets.Repository, options KustomizeProviderOptions) (targets.FileList, error) {
	original := make(map[string][]byte)
	files := make(map[string]map[string]interface{})
	for _, update := range options.Updates {
		if _, ok := files[update.KustomizationFile]; !ok {
			file, err := repository.Get(update.KustomizationFile, options.Ref)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve %s from repository: %w", update.KustomizationFile, err)
			}

			original[update.KustomizationFile] = file
			files[update.KustomizationFile] = make(map[string]interface{})
			if err := yaml.Unmarshal(file, files[update.KustomizationFile]); err != nil {
				return nil, fmt.Errorf("could not parse YAML file %s: %w", update.KustomizationFile, err)
			}
		}

		values := files[update.KustomizationFile]
		images, ok := values["images"]
		if !ok {
			images = []interface{}{}
		}

		// Make sure existing value is an array
		imageList, ok := images.([]interface{})
		if !ok {
			return nil, fmt.Errorf("kustomization file .images field is not an array")
		}

		// Check for existing entries
		found := false
		for index := range imageList {
			current, ok := imageList[index].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("found invalid entry in image list")
			}
			if current["name"] == update.Image {
				if update.NewImage != "" {
					current["newImage"] = update.NewImage
				}
				if update.NewTag != "" {
					current["newTag"] = update.NewTag
				}
				found = true
				break
			}
		}
		if !found {
			newEntry := map[string]interface{}{
				"name": update.Image,
			}
			if update.NewImage != "" {
				newEntry["newImage"] = update.NewImage
			}
			if update.NewTag != "" {
				newEntry["newTag"] = update.NewTag
			}
			imageList = append(imageList, newEntry)
		}
		values["images"] = imageList
		files[update.KustomizationFile] = values
	}

	diff := make(targets.FileList)
	for file, values := range files {
		byt, err := yaml.Marshal(values)
		if err != nil {
			return nil, fmt.Errorf("could not serialize modified file to YAML: %w", err)
		}

		// Skip if there are no changes
		if bytes.Equal(original[file], byt) {
			continue
		}

		diff[file] = byt
	}

	return diff, nil
}
