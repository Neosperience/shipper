package helm_templater_test

import (
	"errors"
	"testing"

	"github.com/neosperience/shipper/patch"
	"github.com/neosperience/shipper/targets"
	helm_templater "github.com/neosperience/shipper/templater/helm"
	"gopkg.in/yaml.v3"
)

const testChart = `replicaCount: 1

envName:

image:
  repository: somerandom.tld/org/name
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: latest

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

podAnnotations: {}

podSecurityContext:
  fsGroup: 2000

service:
  type: ClusterIP
  port: 3000
  proxyPort: 4180

metricsService:
  type: ClusterIP
  port: 9464
  proxyPort: 4181

ingress:
  enabled: true
  annotations:
    cert-manager.io/cluster-issuer: ssl-issuer
    nginx.ingress.kubernetes.io/enable-cors: "true"
  host:

resources: {}

env:
  LOG_LEVEL: info

secrets:
  provider: secretsManager
  names: []

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80

nodeSelector: {}

tolerations: []

affinity: {}
`

func testUpdateHelmChart(t *testing.T, file string) {
	newImage := "git.org/myorg/myrepo"
	newTag := "2022-02-22"

	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/values.yaml": []byte(file),
	})

	commitData, err := helm_templater.UpdateHelmChart(repo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "path/to/values.yaml",
		ImagePath:  "image.repository",
		Image:      newImage,
		TagPath:    "image.tag",
		Tag:        newTag,
	})
	if err != nil {
		t.Fatalf("Failed updating values.yaml: %s", err)
	}

	val, ok := commitData["path/to/values.yaml"]
	if !ok {
		t.Fatal("Expected values.yaml to be modified but wasn't found")
	}

	// Re-parse YAML
	var parsed struct {
		Image struct {
			Repository string `yaml:"repository"`
			Tag        string `yaml:"tag"`
		} `yaml:"image"`
	}
	err = yaml.Unmarshal(val, &parsed)
	if err != nil {
		t.Fatalf("Failed to parse committed YAML: %s", err.Error())
	}
	if parsed.Image.Repository != newImage {
		t.Fatal("Image repository was not set to the new expected value")
	}
	if parsed.Image.Tag != newTag {
		t.Fatal("Image tag was not set to the new expected value")
	}
}

func TestUpdateHelmChartExisting(t *testing.T) {
	testUpdateHelmChart(t, testChart)
}

func TestUpdateHelmChartEmpty(t *testing.T) {
	testUpdateHelmChart(t, ``)
}

func TestUpdateHelmChartFaultyRepository(t *testing.T) {
	brokenrepo := targets.NewInMemoryRepository(targets.FileList{
		"non-yaml/values.yaml":     []byte{0xff, 0xd8, 0xff, 0xe0},
		"broken-image/values.yaml": []byte("image: 12"),
		"broken-tag/values.yaml":   []byte("tag: 12"),
	})

	// Test with inexistant file
	_, err := helm_templater.UpdateHelmChart(brokenrepo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "inexistant-path/values.yaml",
		ImagePath:  "image.repository",
		Image:      "test",
		TagPath:    "image.tag",
		Tag:        "test",
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file did not exist!")
	} else {
		if !errors.Is(err, targets.ErrFileNotFound) {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
	}

	// Test with non-YAML file
	_, err = helm_templater.UpdateHelmChart(brokenrepo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "non-yaml/values.yaml",
		ImagePath:  "image.repository",
		Image:      "test",
		TagPath:    "image.tag",
		Tag:        "test",
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file is not a YAML file!")
	}

	// Test with invalid image path
	_, err = helm_templater.UpdateHelmChart(brokenrepo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "broken-image/values.yaml",
		ImagePath:  "image.name",
		Image:      "test",
		TagPath:    "tag.name",
		Tag:        "test",
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file has an invalid format!")
	} else {
		if !errors.Is(err, patch.ErrInvalidYAMLStructure) {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
	}

	// Test with invalid tag path
	_, err = helm_templater.UpdateHelmChart(brokenrepo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "broken-tag/values.yaml",
		ImagePath:  "image.name",
		Image:      "test",
		TagPath:    "tag.name",
		Tag:        "test",
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file has an invalid format!")
	} else {
		if !errors.Is(err, patch.ErrInvalidYAMLStructure) {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
	}
}

func TestUpdateHelmChartNoChanges(t *testing.T) {
	file := `image:
    pullPolicy: IfNotPresent
    repository: somerandom.tld/org/name
    tag: latest
`
	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/values.yaml": []byte(file),
	})

	commitData, err := helm_templater.UpdateHelmChart(repo, helm_templater.HelmProviderOptions{
		Ref:        "main",
		ValuesFile: "path/to/values.yaml",
		ImagePath:  "image.repository",
		Image:      "somerandom.tld/org/name",
		TagPath:    "image.tag",
		Tag:        "latest",
	})
	if err != nil {
		t.Fatalf("Failed updating values.yaml: %s", err)
	}

	if len(commitData) > 0 {
		t.Fatalf("Found %d changes but commit was expected to be empty", len(commitData))
	}
}
