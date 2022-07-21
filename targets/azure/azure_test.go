package azure_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
	"github.com/neosperience/shipper/test"
)

func TestCommit(t *testing.T) {
	testUser := "test-user@org.tld"
	testKey := "test-key"
	push := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	test.MustSucceed(t, push.Files.Add(map[string][]byte{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	}), "Failed adding test files")

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Mock ref request as well
		if req.Method != "POST" {
			_ = jsoniter.ConfigFastest.NewEncoder(rw).Encode(refList{
				Value: []azureRef{{
					Name:     "refs/heads/" + push.Branch,
					ObjectID: "test-object-id",
				}},
				Count: 1,
			})
			return
		}

		// Check authorization key
		user, key, ok := req.BasicAuth()
		if !ok {
			t.Fatal("Invalid or empty HTTP Basic auth header received")
		}
		test.AssertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		test.AssertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		// Decode payload
		var payload pushData
		err := jsoniter.ConfigFastest.NewDecoder(req.Body).Decode(&payload)
		test.MustSucceed(t, err, "Failed to decode payload")
		defer req.Body.Close()

		test.AssertExpected(t, len(payload.Commits), 1, "Expected 1 commit")
		test.AssertExpected(t, len(payload.RefUpdates), 1, "Expected 1 ref update")
		test.AssertExpected(t, payload.RefUpdates[0].Name, "refs/heads/"+push.Branch, "Expected ref update branch is not correct")
		test.AssertExpected(t, payload.RefUpdates[0].OldObjectID, "test-object-id", "Expected ref update object id is not correct")

		// Make sure files have been encoded with the most efficient encoding
		for _, change := range payload.Commits[0].Changes {
			switch change.Item.Path {
			case "/textfile.txt":
				test.AssertExpected(t, change.NewContent.ContentType, "rawtext", "Expected text encoding for textfile.txt")
				test.AssertExpected(t, change.NewContent.Content, string(push.Files["textfile.txt"]), "Content for textfile.txt doesn't match expected value")
			case "/binaryfile.jpg":
				test.AssertExpected(t, change.NewContent.ContentType, "base64encoded", "Expected base64 encoding for binaryfile.jpg")
				test.AssertExpected(t, change.NewContent.Content, base64.StdEncoding.EncodeToString(push.Files["binaryfile.jpg"]), "Content for binaryfile.jpg is different than expected")
			default:
				t.Fatal("Unexpected file path in commit: ", change.Item.Path)
			}
		}
		_ = jsoniter.ConfigFastest.NewEncoder(rw).Encode(pushResponse{
			Commits:    []commit{{CommitID: "test-commit-id"}},
			Repository: repository{WebURL: "test"},
		})
	}))
	defer server.Close()
	target := NewAPIClient("test-url", "test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.baseURI = server.URL
	target.client = server.Client()

	test.MustSucceed(t, target.Commit(push), "Failed to commit")
}

func TestGet(t *testing.T) {
	testUser := "testUser@org.tld"
	testKey := "test-key"
	testPath := "path/to/test-key"
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
		test.MustSucceed(t, err, "Failed to write test data")
	}))
	defer server.Close()
	target := NewAPIClient("test-project", "test-repository", fmt.Sprintf("%s:%s", testUser, testKey))
	target.baseURI = server.URL
	target.client = server.Client()

	byt, err := target.Get(testPath, "main")
	test.MustSucceed(t, err, "Failed to get file")
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

	// Test with faulty server
	target := NewAPIClient(server.URL, "test-project", "unused")
	target.client = server.Client()

	_, err := target.Get("test", "main")
	test.MustFail(t, err, "Request supposed to error out but Get call exited successfully")

	err = target.Commit(payload)
	test.MustFail(t, err, "Request supposed to error out but Commit call exited successfully")

	// Test with unreachable target
	target = NewAPIClient("http://0.0.0.0", "test-project", "unused")
	target.client = server.Client()
	target.client.Timeout = time.Millisecond // Set a low timeout since we don't want this to work anyway

	_, err = target.Get("test", "main")
	test.MustFail(t, err, "Request supposed to error out but Get call exited successfully")

	err = target.Commit(payload)
	test.MustFail(t, err, "Request supposed to error out but Commit call exited successfully")
}

func TestFaultyCommit(t *testing.T) {
	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Mock missing ref
		if req.Method != "POST" {
			test.MustSucceed(t, jsoniter.ConfigFastest.NewEncoder(rw).Encode(refList{
				Value: []azureRef{},
				Count: 0,
			}), "Failed to encode ref list")
			return
		}
	}))
	defer server.Close()

	target := NewAPIClient("test-url", "test-project", "test-user@org.tld:test-key")
	target.baseURI = server.URL
	target.client = server.Client()

	push := targets.NewPayload("test-branch", "test-author <author@example.com>", "Hello")
	test.MustSucceed(t, push.Files.Add(map[string][]byte{
		"textfile.txt": []byte("test file"),
	}), "Failed adding test files")
	test.MustFail(t, target.Commit(push), "Commit supposed to fail for missing ref but succeeded")
}
