package patch

import (
	"testing"

	"github.com/neosperience/shipper/test"
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
			Value: "to-change",
		},
	}

	byt, err := yaml.Marshal(randomStruct)
	test.MustSucceed(t, err, "YAML encoding failed")

	asMap := make(map[string]any)
	err = yaml.Unmarshal(byt, asMap)
	test.MustSucceed(t, err, "YAML decoding failed")

	// Assert that decoding went fine
	test.AssertExpected(t, asMap["nested"].(map[string]any)["value"].(string), "to-change", ".Nested.Value value is different than expected (initial value)")

	// Call SetPath on an existing tree
	err = SetPath(asMap, "nested.value", "changed")
	test.MustSucceed(t, err, "Failed to set value")

	// Check that value was changed
	test.AssertExpected(t, asMap["nested"].(map[string]any)["value"].(string), "changed", ".Nested.Value value is different than expected (modified value)")

	// Call SetPath on a new tree
	err = SetPath(asMap, "nested.other-value", "new-value")
	test.MustSucceed(t, err, "Failed to set value")

	// Check that value was changed
	test.AssertExpected(t, asMap["nested"].(map[string]any)["other-value"].(string), "new-value", ".Nested.Other-Value value is different than expected (modified value)")
}

func TestSetPathEmpty(t *testing.T) {
	asMap := make(map[string]any)

	// Call SetPath on an existing tree
	err := SetPath(asMap, "nested.value", "changed")
	test.MustSucceed(t, err, "SetPath should not fail on empty map")

	// Check that value was changed
	test.AssertExpected(t, asMap["nested"].(map[string]any)["value"].(string), "changed", ".Nested.Value value is different than expected")
}

func TestSetPathInvalid(t *testing.T) {
	asMap := make(map[string]any)
	asMap["nested"] = 12

	// Call SetPath on an existing tree
	err := SetPath(asMap, "nested.value", "changed")
	switch err {
	case nil:
		t.Fatal("SetPath should fail when types don't match")
	case ErrInvalidYAMLStructure:
		// OK
	default:
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}

func BenchmarkSetPath(b *testing.B) {
	asMap := make(map[string]any)
	err := SetPath(asMap, "nested.value", "new-value")
	test.MustSucceed(b, err, "Failed to set initial value")

	for n := 0; n < b.N; n++ {
		test.MustSucceed(b, SetPath(asMap, "nested.value", n), "Failed to set value")
	}
}
