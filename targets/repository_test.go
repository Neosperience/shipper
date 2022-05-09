package targets_test

import (
	"bytes"
	"testing"

	"github.com/neosperience/shipper/targets"
	"github.com/neosperience/shipper/test"
)

func TestInMemoryRepository(t *testing.T) {
	// Initialize repo
	store := targets.FileList{
		"testfile.txt": []byte("test-data"),
	}
	repo := targets.NewInMemoryRepository(store)

	// Get test file
	data, err := repo.Get("testfile.txt", "dummyref")
	test.MustSucceed(t, err, "Failed getting test file")
	if !bytes.Equal(data, store["testfile.txt"]) {
		t.Fatal("retrieved data for test file doesn't match initial file content")
	}

	// Check for inexistant entry
	_, err = repo.Get("__dummyfile", "dummyref")
	if err == nil {
		t.Fatal("found nonexistent file")
	}

	// Submit new data
	payload := targets.CommitPayload{
		Files: targets.FileList{
			"testfile.txt": []byte("newcontent"),
			"newfile.txt":  []byte("hello im new"),
		},
	}
	test.MustSucceed(t, repo.Commit(&payload), "Failed committing new data")

	// Retrieve modified file
	data, err = repo.Get("testfile.txt", "dummyref")
	test.MustSucceed(t, err, "Failed getting test file")
	if !bytes.Equal(data, payload.Files["testfile.txt"]) {
		t.Fatal("retrieved data for test file doesn't match modified file content")
	}

	// Retrieve new file
	data, err = repo.Get("newfile.txt", "dummyref")
	test.MustSucceed(t, err, "Failed getting new file")
	if !bytes.Equal(data, payload.Files["newfile.txt"]) {
		t.Fatal("retrieved data for new file doesn't match file content")
	}
}
