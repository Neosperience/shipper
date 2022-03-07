package gitea_target

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
)

func assertExpected(t *testing.T, val string, expected string, errmsg string) {
	if val != expected {
		t.Fatalf("%s [expected=\"%s\" | got=\"%s\"]", errmsg, expected, val)
	}
}

func TestCommit(t *testing.T) {
	testUser := "test-user"
	testKey := "test-key"
	commit := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	commit.Files.Add(map[string][]byte{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	})

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
		assertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		assertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		parts := strings.Split(req.URL.Path, "/")
		file := parts[len(parts)-1]

		// If GET-ting, we're asking for the file info
		if req.Method == http.MethodGet {
			sha, ok := hashes[file]
			if !ok {
				http.Error(rw, "not found", http.StatusNotFound)
				return
			}

			if err := jsoniter.ConfigFastest.NewEncoder(rw).Encode(struct {
				SHA string `json:"sha"`
			}{
				SHA: sha,
			}); err != nil {
				t.Fatalf("Failed sending SHA info: %s", err.Error())
			}
			return
		}

		// Must be put PUT-ting
		var payload CommitData
		err := jsoniter.ConfigFastest.NewDecoder(req.Body).Decode(&payload)
		if err != nil {
			t.Fatalf("Failed decoding request payload: %s", err.Error())
		}

		byt, err := base64.StdEncoding.DecodeString(payload.Content)
		if err != nil {
			t.Fatalf("Failed decoding base64-encoded file content: %s", err.Error())
		}

		if !bytes.Equal(byt, commit.Files[file]) {
			t.Fatal("Decoded file content doesn't match expected")
		}

		_ = jsoniter.ConfigFastest.NewEncoder(rw).Encode(struct {
			Commit interface{} `json:"commit"`
		}{
			Commit: struct {
				HTMLURL string `json:"html_url"`
			}{
				HTMLURL: "https://github.com/test-user/test-repo/commit/testsha",
			},
		})
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()

	if err := target.Commit(commit); err != nil {
		t.Fatal(err.Error())
	}
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
		assertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		assertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		rw.Write(testData)
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()

	byt, err := target.Get(testKey, "main")
	if err != nil {
		t.Fatal(err.Error())
	}
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
	payload.Files.Add(map[string][]byte{
		"textfile.txt": []byte("test file"),
	})

	// Test with erroring server
	target := NewAPIClient(server.URL, "test-project", "unused")
	target.client = server.Client()

	if _, err := target.Get("test", "main"); err == nil {
		t.Fatal("Request supposed to error out but Get call exited successfully")
	}

	if err := target.Commit(payload); err == nil {
		t.Fatal("Request supposed to error out but Commit call exited successfully")
	}

	// Test with unreacheable target
	target = NewAPIClient("http://0.0.0.0", "test-project", "unused")
	target.client = server.Client()
	target.client.Timeout = time.Millisecond // Set a low timeout since we don't want this to work anyway

	if _, err := target.Get("test", "main"); err == nil {
		t.Fatal("Request supposed to error out but Get call exited successfully")
	}

	if err := target.Commit(payload); err == nil {
		t.Fatal("Request supposed to error out but Commit call exited successfully")
	}
}
