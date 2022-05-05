package test

import (
	"testing"
)

// AssertExpected checks that a value is as expected and fails the test if it is not.
func AssertExpected[T comparable](t *testing.T, val T, expected T, errorMessage string) {
	if val != expected {
		t.Fatalf("%s [expected=\"%v\" | got=\"%v\"]", errorMessage, expected, val)
	}
}
