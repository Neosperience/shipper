package kustomize_templater_test

import (
	"testing"

	"github.com/neosperience/shipper/targets"
	kustomize_templater "github.com/neosperience/shipper/templater/kustomize"
	"gopkg.in/yaml.v3"
)

const kustomizationEmpty = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: nspod
resources:
- ../base
- ingress.yml
- secrets.yml

`

const kustomizationExisting = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: nspod
resources:
- ../base
- ingress.yml
- secrets.yml

configMapGenerator:
- literals:
  - LOG_LEVEL=debug
  name: svc-config

images:
- name: git.org/myorg/myrepo
  newTag: bbaaff

replicas:
  - name: svc
    count: 2
`

func testUpdateKustomizationCommon(t *testing.T, kustomizeFile string) {
	newImage := "git.org/myorg/myrepo"
	newTag := "2022-02-22"

	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/kustomization.yaml": []byte(kustomizeFile),
	})

	commitData, err := kustomize_templater.UpdateKustomization(repo, kustomize_templater.KustomizeProviderOptions{
		Ref:               "dummy",
		KustomizationFile: "path/to/kustomization.yaml",
		Image:             newImage,
		NewTag:            newTag,
	})
	if err != nil {
		t.Fatalf("Failed updating kustomization.yaml: %s", err.Error())
	}

	val, ok := commitData["path/to/kustomization.yaml"]
	if !ok {
		t.Fatal("Expected kustomization.yaml to be modified but wasn't found")
	}

	var partial struct {
		Image []struct {
			Name   string `yaml:"name"`
			NewTag string `yaml:"newTag"`
		} `yaml:"images"`
	}
	err = yaml.Unmarshal(val, &partial)
	if err != nil {
		t.Fatalf("Failed to unmarshal committed file: %s", err.Error())
	}

	if len(partial.Image) < 1 {
		t.Fatal("No image fields were found")
	}

	if partial.Image[0].Name != newImage {
		t.Fatal("Image name is different than expected")
	}
	if partial.Image[0].NewTag != newTag {
		t.Fatal("Image tag is different than expected")
	}
}

func TestUpdateKustomizationNoImages(t *testing.T) {
	testUpdateKustomizationCommon(t, kustomizationEmpty)
}

func TestUpdateKustomizationEmpty(t *testing.T) {
	testUpdateKustomizationCommon(t, ``)
}

func TestUpdateKustomizationExisting(t *testing.T) {
	testUpdateKustomizationCommon(t, kustomizationExisting)
}
