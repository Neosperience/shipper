package patch

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSetPathExisting(t *testing.T) {
	randomStruct := struct {
		Test   string
		Nested struct {
			Value string
		}
	}{
		Test: "random-value",
		Nested: struct {
			Value string
		}{
			Value: "tochange",
		},
	}

	byt, err := yaml.Marshal(randomStruct)
	if err != nil {
		t.Fatalf("YAML encoding failed: %s", err.Error())
	}

	asMap := make(map[string]interface{})
	err = yaml.Unmarshal(byt, asMap)
	if err != nil {
		t.Fatalf("YAML decoding failed: %s", err.Error())
	}

	// Assert that decoding went fine
	if asMap["nested"].(map[string]interface{})["value"] != "tochange" {
		t.Fatal("Expected .Nested.Value value is different")
	}

	// Call SetPath on an existing tree
	err = SetPath(asMap, "nested.value", "changed")
	if err != nil {
		t.Fatal(err)
	}

	// Check that value was changed
	if asMap["nested"].(map[string]interface{})["value"] != "changed" {
		t.Fatal("Expected .Nested.Value value is different")
	}

	// Call SetPath on a new tree
	err = SetPath(asMap, "nested.other-value", "new-value")
	if err != nil {
		t.Fatal(err)
	}

	// Check that value was changed
	if asMap["nested"].(map[string]interface{})["other-value"] != "new-value" {
		t.Fatal("New value not found or different")
	}
}

func TestSetPathEmpty(t *testing.T) {
	asMap := make(map[string]interface{})

	// Call SetPath on an existing tree
	err := SetPath(asMap, "nested.value", "changed")
	if err != nil {
		t.Fatal(err)
	}

	// Check that value was changed
	if asMap["nested"].(map[string]interface{})["value"] != "changed" {
		t.Fatal("Expected .Nested.Value value is different")
	}
}

func TestSetPathInvalid(t *testing.T) {
	asMap := make(map[string]interface{})
	asMap["nested"] = 12

	// Call SetPath on an existing tree
	err := SetPath(asMap, "nested.value", "changed")
	if err == nil {
		t.Fatal("expected error but everything went fine")
	} else {
		if err != ErrInvalidYAMLStructure {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	}
}

func BenchmarkSetPath(b *testing.B) {
	asMap := make(map[string]interface{})
	err := SetPath(asMap, "nested.value", "new-value")
	if err != nil {
		b.Fatal("Failed to set initial value")
	}

	for n := 0; n < b.N; n++ {
		SetPath(asMap, "nested.value", n)
	}
}
