package targets

import "fmt"

// Repository is a supported platform where we can push commits to
type Repository interface {
	// Get retrieves a file from the repository
	Get(path string, ref string) ([]byte, error)

	// Commit creates a commit from a payload and pushes it to the repository
	Commit(data *CommitPayload) error
}

// InMemoryRepository is an in-memory implementation of Repository for testing
type InMemoryRepository struct {
	Files FileList
}

// NewInMemoryRepository creates a new InMemoryRepository
func NewInMemoryRepository(files FileList) *InMemoryRepository {
	return &InMemoryRepository{
		Files: files,
	}
}

func (m *InMemoryRepository) Get(path string, ref string) ([]byte, error) {
	// ref is ignored
	file, ok := m.Files[path]
	if !ok {
		return nil, fmt.Errorf("file not found")
	}

	return file, nil
}

func (m *InMemoryRepository) Commit(data *CommitPayload) error {
	for name, content := range data.Files {
		m.Files[name] = content
	}
	return nil
}
