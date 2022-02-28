package gitlab_target

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
)

func TestCommit(t *testing.T) {
	testKey := "test-key"
	commit := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	commit.Files.Add(map[string][]byte{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	})

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		token := req.Header.Get("PRIVATE-TOKEN")
		if token != testKey {
			t.Fatalf("Expected '%s' as PRIVATE-TOKEN, got '%s'", testKey, token)
		}

		// Decode payload
		var payload CommitPostData
		err := jsoniter.ConfigFastest.NewDecoder(req.Body).Decode(&payload)
		if err != nil {
			t.Fatal(err)
		}
		defer req.Body.Close()

		// Make sure files have been encoded with the most efficient encoding
		for _, action := range payload.Actions {
			switch action.FilePath {
			case "textfile.txt":
				entry := commit.Files[action.FilePath]
				if action.Encoding != "text" {
					t.Error("text file has not been encoded with text encoding")
				}
				if action.Content != string(entry) {
					t.Error("text file content doesnt match")
				}
			case "binaryfile.jpg":
				entry := commit.Files[action.FilePath]
				if action.Encoding != "base64" {
					t.Error("binary file has not been encoded with b64 encoding")
				}
				if action.Content != base64.StdEncoding.EncodeToString(entry) {
					t.Error("binary file content doesnt match")
				}
			}
		}
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", testKey)
	target.client = server.Client()

	if err := target.Commit(commit); err != nil {
		t.Fatal(err.Error())
	}
}

func TestGet(t *testing.T) {
	testKey := "path/to/test-key"
	testData := []byte("hello test here")
	testDataHash := sha256.Sum256(testData)

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		token := req.Header.Get("PRIVATE-TOKEN")
		if token != testKey {
			t.Fatalf("Expected '%s' as PRIVATE-TOKEN, got '%s'", testKey, token)
		}

		parts := strings.Split(testKey, "/")
		baseKey := parts[len(parts)-1]

		err := jsoniter.ConfigFastest.NewEncoder(rw).Encode(FileInfo{
			FileName:      baseKey,
			FilePath:      testKey,
			Size:          len(testData),
			Encoding:      "base64",
			Content:       base64.StdEncoding.EncodeToString(testData),
			ContentSha256: hex.EncodeToString(testDataHash[:]),
			Ref:           "main",
			BlobID:        "somerandomid",
			CommitID:      "someotherid",
		})
		if err != nil {
			t.Fatalf("Failed encoding response: %s", err.Error())
		}
	}))
	defer server.Close()
	target := NewAPIClient(server.URL, "test-project", testKey)
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
