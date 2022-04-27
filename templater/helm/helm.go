package helm_templater

import (
	"bytes"
	"fmt"

	"github.com/neosperience/shipper/patch"
	"github.com/neosperience/shipper/targets"
	"gopkg.in/yaml.v3"
)

type HelmUpdate struct {
	ValuesFile string
	Image      string
	ImagePath  string
	Tag        string
	TagPath    string
}

type HelmProviderOptions struct {
	Ref     string
	Updates []HelmUpdate
}

func UpdateHelmChart(repository targets.Repository, options HelmProviderOptions) (targets.FileList, error) {
	original := make(map[string][]byte)
	files := make(map[string]map[string]interface{})
	for _, update := range options.Updates {
		if _, ok := files[update.ValuesFile]; !ok {
			file, err := repository.Get(update.ValuesFile, options.Ref)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve %s from repository: %w", update.ValuesFile, err)
			}

			original[update.ValuesFile] = file
			files[update.ValuesFile] = make(map[string]interface{})
			if err := yaml.Unmarshal(file, files[update.ValuesFile]); err != nil {
				return nil, fmt.Errorf("could not parse YAML file %s: %w", update.ValuesFile, err)
			}
		}

		if err := patch.SetPath(files[update.ValuesFile], update.ImagePath, update.Image); err != nil {
			return nil, fmt.Errorf("could not patch image for %s: %w", update.ValuesFile, err)
		}
		if err := patch.SetPath(files[update.ValuesFile], update.TagPath, update.Tag); err != nil {
			return nil, fmt.Errorf("could not patch image for %s: %w", update.ValuesFile, err)
		}
	}

	diff := make(targets.FileList)
	for file, content := range files {
		byt, err := yaml.Marshal(content)
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
