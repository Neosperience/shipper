package gitlab_target

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"gitlab.neosperience.com/tools/shipper/targets"
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
