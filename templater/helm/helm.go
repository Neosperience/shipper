package helm_templater

import (
	"bytes"
	"fmt"

	"gitlab.neosperience.com/tools/shipper/patch"
	"gitlab.neosperience.com/tools/shipper/targets"
	"gopkg.in/yaml.v3"
)

type HelmProviderOptions struct {
	Ref        string
	ValuesFile string
	Image      string
	ImagePath  string
	Tag        string
	TagPath    string
}

func UpdateHelmChart(repository targets.Repository, options HelmProviderOptions) (targets.FileList, error) {
	file, err := repository.Get(options.ValuesFile, options.Ref)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve values.yaml from repository: %w", err)
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(file, values); err != nil {
		return nil, fmt.Errorf("could not parse values.yaml: %w", err)
	}
	if err := patch.SetPath(values, options.ImagePath, options.Image); err != nil {
		return nil, fmt.Errorf("could not patch image for values.yaml: %w", err)
	}
	if err := patch.SetPath(values, options.TagPath, options.Tag); err != nil {
		return nil, fmt.Errorf("could not patch image for values.yaml: %w", err)
	}

	byt, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("could not serialize modified file to YAML: %w", err)
	}

	if bytes.Equal(byt, file) {
		return targets.FileList{}, nil
	}

	diff := make(targets.FileList)
	diff[options.ValuesFile] = byt

	return diff, nil
}
