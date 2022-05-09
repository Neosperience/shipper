package targets_test

import (
	"fmt"
	"testing"

	"github.com/neosperience/shipper/targets"
	"github.com/neosperience/shipper/test"
)

func TestSplitAuthor(t *testing.T) {
	authorName, authorMail := "test-author", "author@example.com"
	authorString := fmt.Sprintf("%s <%s>", authorName, authorMail)

	// Check a normal "name <mail>" tuple
	commit := targets.NewPayload("", authorString, "")

	name, email := commit.SplitAuthor()
	test.AssertExpected(t, name, authorName, "author name doesn't match expected value")
	test.AssertExpected(t, email, authorMail, "author email doesn't match expected value")

	// Check only name or email
	commit = targets.NewPayload("", authorName, "")
	name, email = commit.SplitAuthor()
	test.AssertExpected(t, name, authorName, "author name doesn't match expected value")
	test.AssertExpected(t, email, "", "author email should be empty")

	commit = targets.NewPayload("", fmt.Sprintf("<%s>", authorMail), "")
	name, email = commit.SplitAuthor()
	test.AssertExpected(t, name, "", "author name should be empty")
	test.AssertExpected(t, email, authorMail, "author email doesn't match expected value")
}

func TestPayloadAdd(t *testing.T) {
	empty := make(targets.FileList)

	files := targets.FileList{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	}

	test.MustSucceed(t, empty.Add(files), "Failed adding test files to empty file list")

	err := empty.Add(files)
	switch err {
	case nil:
		t.Fatal("expected error when adding duplicate files but got none")
	case targets.ErrFileAlreadyAdded:
		// OK
	default:
		t.Fatal("expected error when adding duplicate files but got unexpected error")
	}
}
