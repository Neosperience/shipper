package github_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
	"github.com/neosperience/shipper/test"
)

func TestCommit(t *testing.T) {
	testUser := "test-user"
	testKey := "test-key"
	commit := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	test.MustSucceed(t, commit.Files.Add(map[string][]byte{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	}), "Failed adding test files")

	hashes := map[string]string{
		"textfile.txt": "testsha",
	}

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		user, key, ok := req.BasicAuth()
		if !ok {
			t.Fatal("Invalid or empty HTTP Basic auth header received")
		}
		test.AssertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		test.AssertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		parts := strings.Split(req.URL.Path, "/")
		file := parts[len(parts)-1]

		// If GET-ting, we're asking for the file info
		if req.Method == http.MethodGet {
			sha, ok := hashes[file]
			if !ok {
				http.Error(rw, "not found", http.StatusNotFound)
				return
			}

			test.MustSucceed(t, jsoniter.ConfigFastest.NewEncoder(rw).Encode(struct {
				SHA string `json:"sha"`
			}{
				SHA: sha,
			}), "Failed sending SHA info")
			return
		}

		// Must be put PUT-ting
		var payload CommitData
		test.MustSucceed(t, jsoniter.ConfigFastest.NewDecoder(req.Body).Decode(&payload), "Failed decoding payload")

		byt, err := base64.StdEncoding.DecodeString(payload.Content)
		test.MustSucceed(t, err, "Failed decoding base64-encoded content")
		if !bytes.Equal(byt, commit.Files[file]) {
			t.Fatal("Decoded file content doesn't match expected")
		}

		test.MustSucceed(t, jsoniter.ConfigFastest.NewEncoder(rw).Encode(struct {
			Commit any `json:"commit"`
		}{
			Commit: struct {
				HTMLURL string `json:"html_url"`
			}{
				HTMLURL: "https://github.com/test-user/test-repo/commit/testsha",
			},
		}), "Failed sending commit info")
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()

	test.MustSucceed(t, target.Commit(commit), "Failed committing files")
}

func TestGet(t *testing.T) {
	testUser := "test-user"
	testKey := "test-key"
	testData := []byte("hello test here")

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		user, key, ok := req.BasicAuth()
		if !ok {
			t.Fatal("Invalid or empty HTTP Basic auth header received")
		}
		test.AssertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		test.AssertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		_, err := rw.Write(testData)
		test.MustSucceed(t, err, "Failed writing test data")
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()

	byt, err := target.Get(testKey, "main")
	test.MustSucceed(t, err, "Failed getting file")
	if !bytes.Equal(byt, testData) {
		t.Fatal("Expected file content is different from retrieved")
	}
}

func TestFaultyServer(t *testing.T) {
	// Mock server that just errors out
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	payload := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	test.MustSucceed(t, payload.Files.Add(map[string][]byte{
		"textfile.txt": []byte("test file"),
	}), "Failed adding test file")

	// Test with erroring server
	target := NewAPIClient(server.URL, "test-project", "unused")
	target.client = server.Client()

	_, err := target.Get("test", "main")
	test.MustFail(t, err, "Request supposed to error out but Get call exited successfully")

	err = target.Commit(payload)
	test.MustFail(t, err, "Request supposed to error out but Commit call exited successfully")

	// Test with unreacheable target
	target = NewAPIClient("http://0.0.0.0", "test-project", "unused")
	target.client = server.Client()
	target.client.Timeout = time.Millisecond // Set a low timeout since we don't want this to work anyway

	_, err = target.Get("test", "main")
	test.MustFail(t, err, "Request supposed to error out but Get call exited successfully")

	err = target.Commit(payload)
	test.MustFail(t, err, "Request supposed to error out but Commit call exited successfully")
}
