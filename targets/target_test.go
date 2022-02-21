package targets_test

import (
	"fmt"
	"testing"

	"gitlab.neosperience.com/tools/shipper/targets"
)

func TestSplitAuthor(t *testing.T) {
	authorName, authorMail := "test-author", "author@example.com"
	authorString := fmt.Sprintf("%s <%s>", authorName, authorMail)

	// Check a normal "name <mail>" tuple
	commit := targets.NewPayload("", authorString, "")

	name, email := commit.SplitAuthor()
	if name != authorName {
		t.Fatalf("expected author name to be '%s', got '%s'", authorName, name)
	}
	if email != authorMail {
		t.Fatalf("expected author email to be '%s', got '%s'", authorMail, email)
	}

	// Check only name or email
	commit = targets.NewPayload("", authorName, "")
	name, email = commit.SplitAuthor()
	if name != authorName {
		t.Fatalf("expected author name to be '%s', got '%s'", authorName, name)
	}
	if email != "" {
		t.Fatalf("expected author email to be empty, got '%s'", email)
	}

	commit = targets.NewPayload("", fmt.Sprintf("<%s>", authorMail), "")
	name, email = commit.SplitAuthor()
	if name != "" {
		t.Fatalf("expected author name to be empty, got '%s'", name)
	}
	if email != authorMail {
		t.Fatalf("expected author email to be '%s', got '%s'", authorMail, email)
	}
}

func TestPayloadAdd(t *testing.T) {
	commit := targets.NewPayload("test", "test", "test")

	files := map[string][]byte{
		"textfile.txt":   []byte("test file"),
		"binaryfile.jpg": {0xff, 0xd8, 0xff, 0xe0},
	}

	err := commit.Add(files)
	if err != nil {
		t.Fatal(err)
	}

	err = commit.Add(files)
	if err == nil {
		t.Fatal("expected error when adding duplicate files but got none")
	} else if err != targets.ErrFileAlreadyAdded {
		t.Fatal(err)
	}
}
