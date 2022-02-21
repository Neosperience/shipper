package targets

import (
	"errors"
	"strings"
)

// CommitPayload is a simplified representation of git commits that can be pushed using a Target
type CommitPayload struct {
	Branch  string
	Author  string
	Message string
	Files   map[string][]byte
}

// Target is a supported platform where we can push commits to
type Target interface {
	// Commit creates a commit from a payload and pushes it to the target
	Commit(data CommitPayload) error
}

var (
	// ErrFileAlreadyAdded happens if we're trying to add a file to a commit payload when one with the same name is already present
	ErrFileAlreadyAdded = errors.New("file already added")
)

// NewPayload creates a new empty commit payload
func NewPayload(branch string, author string, message string) *CommitPayload {
	return &CommitPayload{
		Branch:  branch,
		Author:  author,
		Message: message,
		Files:   make(map[string][]byte),
	}
}

// Add adds files to a commit payload
func (payload *CommitPayload) Add(files map[string][]byte) error {
	// Check if we are ok to merge
	for name := range files {
		_, ok := payload.Files[name]
		if ok {
			return ErrFileAlreadyAdded
		}
	}

	// Merge maps
	for name, content := range files {
		payload.Files[name] = content
	}

	return nil
}

// SplitAuthor splits the Author field to return a tuple of (name, email) fields
// Since the field is quite dynamic, either field could be empty
func (payload *CommitPayload) SplitAuthor() (string, string) {
	// Search for separator (<) in "name <email>"
	emailSeparator := strings.IndexRune(payload.Author, '<')

	// No separator? Assume name-only
	if emailSeparator < 0 {
		return payload.Author, ""
	}

	// Split and trim out the "<>"s
	name, email := payload.Author[:emailSeparator], strings.Trim(payload.Author[emailSeparator:], "<>")

	return strings.TrimSpace(name), strings.TrimSpace(email)
}
