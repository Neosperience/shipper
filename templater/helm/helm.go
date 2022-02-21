package helm_templater

import (
	"fmt"

	"gitlab.neosperience.com/tools/shipper/targets"
)

type HelmProviderOptions struct {
	ValuesFile string
	Image      string
	ImagePath  string
	Tag        string
	TagPath    string
}

func UpdateHelmChart(options HelmProviderOptions) (targets.FileList, error) {

	return targets.FileList{}, fmt.Errorf("not implemented")
}
