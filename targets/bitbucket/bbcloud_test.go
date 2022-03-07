package bitbucket_target

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	// Setup test HTTP server/client
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Check authorization key
		user, key, ok := req.BasicAuth()
		if !ok {
			t.Fatal("Invalid or empty HTTP Basic auth header received")
		}
		assertExpected(t, user, testUser, "Basic auth user doesn't match expected value")
		assertExpected(t, key, testKey, "Basic auth password doesn't match expected value")

		err := req.ParseForm()
		if err != nil {
			t.Fatal("Could not parse formdata")
		}
		assertExpected(t, req.Form.Get("author"), commit.Author, "Form author doesn't match expected commit author")
		assertExpected(t, req.Form.Get("branch"), commit.Branch, "Form branch doesn't match expected commit branch")
		assertExpected(t, req.Form.Get("message"), commit.Message, "Form message doesn't match expected commit message")

		rw.Header().Set("Location", "https://bitbucket.org/test-user/test-repo/commits/test-commit")
	}))
	defer server.Close()
	target := NewCloudAPIClient("test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()
	target.baseURI = server.URL

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
	target := NewCloudAPIClient("test-project", fmt.Sprintf("%s:%s", testUser, testKey))
	target.client = server.Client()
	target.baseURI = server.URL

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
