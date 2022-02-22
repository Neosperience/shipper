package helm_templater_test

import (
	"testing"

	"gitlab.neosperience.com/tools/shipper/targets"
	helm_templater "gitlab.neosperience.com/tools/shipper/templater/helm"
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

func TestUpdateHelmChart(t *testing.T) {
	newImage := "git.org/myorg/myrepo"
	newTag := "2022-02-22"

	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/values.yaml": []byte(testChart),
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
