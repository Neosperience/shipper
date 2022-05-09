package json_templater_test

import (
	"errors"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
	"github.com/neosperience/shipper/test"

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
	test.MustSucceed(t, err, "Failed updating cdk.json")

	val, ok := commitData["path/to/cdk.json"]
	if !ok {
		t.Fatal("Expected cdk.json to be modified but wasn't found")
	}

	// Re-parse JSON
	var parsed struct {
		Image string `json:"image.env=build"`
	}
	test.MustSucceed(t, jsoniter.Unmarshal(val, &parsed), "Failed parsing cdk.json")
	test.AssertExpected(t, parsed.Image, newTag, "Image tag was not set to the new expected value")
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
	switch {
	case err == nil:
		t.Fatal("Updating repo succeeded but the original file did not exist!")
	case errors.Is(err, targets.ErrFileNotFound):
		// Expected
	default:
		t.Fatalf("Unexpected error: %s", err)
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
	test.MustSucceed(t, err, "Failed updating cdk.json")
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
	test.MustSucceed(t, err, "Failed updating JSON files")

	valuesFile, ok := commitData["path/to/cdk.json"]
	if !ok {
		t.Fatal("Failed to find cdk.json in commit data")
	}

	var parsedValues struct {
		Image string `json:"image"`
	}
	test.MustSucceed(t, jsoniter.Unmarshal(valuesFile, &parsedValues), "Failed parsing cdk.json")
	test.AssertExpected(t, parsedValues.Image, updates[0].Tag, "cdk.json/image tag was not set to the new expected value")

	otherValuesFile, ok := commitData["path/to/other-values.json"]
	if !ok {
		t.Fatal("Failed to find other-values.json in commit data")
	}
	test.MustSucceed(t, jsoniter.Unmarshal(otherValuesFile, &parsedValues), "Failed parsing other-values.json")
	test.AssertExpected(t, parsedValues.Image, updates[1].Tag, "other-values.json/image tag was not set to the new expected value")
}
