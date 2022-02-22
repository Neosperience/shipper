package kustomize_templater

import (
	"bytes"
	"fmt"

	"gitlab.neosperience.com/tools/shipper/targets"
	"gopkg.in/yaml.v3"
)

type KustomizeProviderOptions struct {
	Ref               string
	KustomizationFile string
	Image             string
	NewImage          string
	NewTag            string
}

func UpdateKustomization(repository targets.Repository, options KustomizeProviderOptions) (targets.FileList, error) {
	file, err := repository.Get(options.KustomizationFile, options.Ref)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve values.yaml from repository: %w", err)
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(file, values); err != nil {
		return nil, fmt.Errorf("could not parse values.yaml: %w", err)
	}

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
		if current["name"] == options.Image {
			if options.NewImage != "" {
				current["newImage"] = options.NewImage
			}
			if options.NewTag != "" {
				current["newTag"] = options.NewTag
			}
			found = true
			break
		}
	}
	if !found {
		newEntry := map[string]interface{}{
			"name": options.Image,
		}
		if options.NewImage != "" {
			newEntry["newImage"] = options.NewImage
		}
		if options.NewTag != "" {
			newEntry["newTag"] = options.NewTag
		}
		imageList = append(imageList, newEntry)
	}
	values["images"] = imageList

	byt, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("could not serialize modified file to YAML: %w", err)
	}

	if bytes.Equal(byt, file) {
		return targets.FileList{}, nil
	}

	diff := make(targets.FileList)
	diff[options.KustomizationFile] = byt

	return diff, nil
}
