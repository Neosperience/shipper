package json_templater_test

import (
	"errors"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"

	json_templater "github.com/neosperience/shipper/templater/json"
)

const testJSON = `{
	"test": "field",
	"nested": {
		"test": "field"
	},
	"fake.nested": "tag",
	"image.env=build": "old"
}`

func testUpdateJSONFile(t *testing.T, file string) {
	imagePath := "image.env=build"
	newTag := "2022-02-22"

	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/cdk.json": []byte(file),
	})

	commitData, err := json_templater.UpdateJSONFile(repo, json_templater.JSONProviderOptions{
		Ref: "main",
		Updates: []json_templater.FileUpdate{
			{
				File: "path/to/cdk.json",
				Path: imagePath,
				Tag:  newTag,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed updating cdk.json: %s", err)
	}

	val, ok := commitData["path/to/cdk.json"]
	if !ok {
		t.Fatal("Expected cdk.json to be modified but wasn't found")
	}

	// Re-parse JSON
	var parsed struct {
		Image string `json:"image.env=build"`
	}
	err = jsoniter.ConfigDefault.Unmarshal(val, &parsed)
	if err != nil {
		t.Fatalf("Failed to parse committed JSON: %s", err.Error())
	}
	if parsed.Image != newTag {
		t.Fatal("Image tag was not set to the new expected value")
	}
}

func TestUpdateJSONFileExisting(t *testing.T) {
	testUpdateJSONFile(t, testJSON)
}

func TestUpdateJSONFileEmpty(t *testing.T) {
	testUpdateJSONFile(t, `{}`)
}

func TestUpdateHelmChartFaultyRepository(t *testing.T) {
	brokenrepo := targets.NewInMemoryRepository(targets.FileList{
		"non-json/cdk.json": []byte{0xff, 0xd8, 0xff, 0xe0},
	})

	// Test with inexistant file
	_, err := json_templater.UpdateJSONFile(brokenrepo, json_templater.JSONProviderOptions{
		Ref: "main",
		Updates: []json_templater.FileUpdate{
			{
				File: "inexistant-path/cdk.json",
				Path: "image",
				Tag:  "test",
			},
		},
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file did not exist!")
	} else {
		if !errors.Is(err, targets.ErrFileNotFound) {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
	}

	// Test with non-JSON file
	_, err = json_templater.UpdateJSONFile(brokenrepo, json_templater.JSONProviderOptions{
		Ref: "main",
		Updates: []json_templater.FileUpdate{
			{
				File: "non-json/cdk.json",
				Path: "image",
				Tag:  "test",
			},
		},
	})
	if err == nil {
		t.Fatal("Updating repo succeeded but the original file is not a JSON file!")
	}
}

func TestUpdateJSONNoChanges(t *testing.T) {
	file := `{
  "image": "latest"
}`
	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/cdk.json": []byte(file),
	})

	commitData, err := json_templater.UpdateJSONFile(repo, json_templater.JSONProviderOptions{
		Ref: "main",
		Updates: []json_templater.FileUpdate{
			{
				File: "path/to/cdk.json",
				Path: "image",
				Tag:  "latest",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed updating cdk.json: %s", err)
	}

	if len(commitData) > 0 {
		t.Fatalf("Found %d changes but commit was expected to be empty", len(commitData))
	}
}

func TestUpdateMultipleImages(t *testing.T) {
	repo := targets.NewInMemoryRepository(targets.FileList{
		"path/to/cdk.json":          []byte(`{"image": "latest"}`),
		"path/to/other-values.json": []byte(`{}`),
	})

	updates := []json_templater.FileUpdate{
		{
			File: "path/to/cdk.json",
			Path: "image",
			Tag:  "latest",
		},
		{
			File: "path/to/other-values.json",
			Path: "image",
			Tag:  "new",
		},
	}

	commitData, err := json_templater.UpdateJSONFile(repo, json_templater.JSONProviderOptions{
		Ref:     "main",
		Updates: updates,
	})
	if err != nil {
		t.Fatalf("Failed updating JSON files: %s", err)
	}

	valuesFile, ok := commitData["path/to/cdk.json"]
	if !ok {
		t.Fatal("Failed to find cdk.json in commit data")
	}

	var parsedValues struct {
		Image string `json:"image"`
	}
	err = jsoniter.ConfigFastest.Unmarshal(valuesFile, &parsedValues)
	if err != nil {
		t.Fatalf("Failed to unmarshal cdk.json: %s", err)
	}
	if parsedValues.Image != updates[0].Tag {
		t.Fatalf("Expected first image to be tagged %s but got %s", updates[0].Tag, parsedValues.Image)
	}

	otherValuesFile, ok := commitData["path/to/other-values.json"]
	if !ok {
		t.Fatal("Failed to find other-values.json in commit data")
	}
	err = jsoniter.ConfigFastest.Unmarshal(otherValuesFile, &parsedValues)
	if err != nil {
		t.Fatalf("Failed to unmarshal other-values.json: %s", err)
	}
	if parsedValues.Image != updates[1].Tag {
		t.Fatalf("Expected second image to be tagged %s but got %s", updates[1].Tag, parsedValues.Image)
	}
}
