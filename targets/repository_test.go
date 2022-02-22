package targets_test

import (
	"bytes"
	"testing"

	"gitlab.neosperience.com/tools/shipper/targets"
)

func TestInMemoryRepository(t *testing.T) {
	// Initialize repo
	store := targets.FileList{
		"testfile.txt": []byte("test-data"),
	}
	repo := targets.NewInMemoryRepository(store)

	// Get test file
	data, err := repo.Get("testfile.txt", "dummyref")
	if err != nil {
		t.Fatalf("failed to find test file: %s", err.Error())
	}
	if !bytes.Equal(data, store["testfile.txt"]) {
		t.Fatal("retrieved data for test file doesn't match initial file content")
	}

	// Check for inexistant entry
	_, err = repo.Get("__dummyfile", "dummyref")
	if err == nil {
		t.Fatal("found inexistant file")
	}

	// Submit new data
	payload := targets.CommitPayload{
		Files: targets.FileList{
			"testfile.txt": []byte("newcontent"),
			"newfile.txt":  []byte("hello im new"),
		},
	}
	err = repo.Commit(&payload)
	if err != nil {
		t.Fatalf("failed to commit test payload: %s", err.Error())
	}

	// Retrieve modified file
	data, err = repo.Get("testfile.txt", "dummyref")
	if err != nil {
		t.Fatalf("failed to find test file: %s", err.Error())
	}
	if !bytes.Equal(data, payload.Files["testfile.txt"]) {
		t.Fatal("retrieved data for test file doesn't match modified file content")
	}

	// Retrieve new file
	data, err = repo.Get("newfile.txt", "dummyref")
	if err != nil {
		t.Fatalf("failed to find new file: %s", err.Error())
	}
	if !bytes.Equal(data, payload.Files["newfile.txt"]) {
		t.Fatal("retrieved data for new file doesn't match file content")
	}
}
