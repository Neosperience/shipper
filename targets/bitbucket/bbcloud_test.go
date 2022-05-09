package bitbucket_target

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		user, key, ok := req.BasicAuth()
		if !ok {
			t.Fatal("Invalid or empty HTTP Basic auth header received")
		}
		test.AssertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		test.AssertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		test.MustSucceed(t, req.ParseForm(), "Failed to parse form")
		test.AssertExpected(t, req.Form.Get("author"), commit.Author, "Form author doesn't match expected commit author")
		test.AssertExpected(t, req.Form.Get("branch"), commit.Branch, "Form branch doesn't match expected commit branch")
		test.AssertExpected(t, req.Form.Get("message"), commit.Message, "Form message doesn't match expected commit message")

		rw.Header().Set("Location", "https://bitbucket.org/test-user/test-repo/commits/test-commit")
	}))
	defer server.Close()
	target := NewCloudAPIClient("test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()
	target.baseURI = server.URL

	test.MustSucceed(t, target.Commit(commit), "Failed to commit")
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
		test.MustSucceed(t, err, "Failed to write test data")
	}))
	defer server.Close()
	target := NewCloudAPIClient("test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()
	target.baseURI = server.URL

	byt, err := target.Get(testKey, "main")
	test.MustSucceed(t, err, "Failed to get test data")
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
	target := NewCloudAPIClient("test-project", "unused")
	target.client = server.Client()
	target.baseURI = server.URL

	if _, err := target.Get("test", "main"); err == nil {
		t.Fatal("Request supposed to error out but Get call exited successfully")
	}

	if err := target.Commit(payload); err == nil {
		t.Fatal("Request supposed to error out but Commit call exited successfully")
	}

	// Test with unreacheable target
	target.baseURI = "http://0.0.0.0"
	target.client.Timeout = time.Millisecond // Set a low timeout since we don't want this to work anyway

	if _, err := target.Get("test", "main"); err == nil {
		t.Fatal("Request supposed to error out but Get call exited successfully")
	}

	if err := target.Commit(payload); err == nil {
		t.Fatal("Request supposed to error out but Commit call exited successfully")
	}
}
