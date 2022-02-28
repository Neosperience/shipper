package github_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
)

// GithubRepository commits to a GitHub repository using the GitHub REST Commits APIs
type GithubRepository struct {
	baseURI     string
	projectID   string
	credentials string

	client *http.Client
}

// NewAPIClient creates a GithubRepository instance
func NewAPIClient(uri string, projectID string, credentials string) *GithubRepository {
	var client = &http.Client{}
	return &GithubRepository{
		baseURI:     uri,
		projectID:   projectID,
		credentials: credentials,
		client:      client,
	}
}

func (gh *GithubRepository) Get(path string, ref string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", gh.baseURI, gh.projectID, path, url.QueryEscape(ref)), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Accept", "application/vnd.github.v3.raw")
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(gh.credentials)))

	res, err := gh.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("request returned error: %s", body)
	}

	return ioutil.ReadAll(res.Body)
}

func (gh *GithubRepository) getFileSHA(path, branch string) (string, bool, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", gh.baseURI, gh.projectID, path, url.QueryEscape(branch)), nil)
	if err != nil {
		return "", false, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(gh.credentials)))

	res, err := gh.client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("error performing request: %w", err)
	}
	defer res.Body.Close()

	// File not found, file does not exist
	if res.StatusCode == 404 {
		return "", false, nil
	}

	// Error encountered
	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		return "", false, fmt.Errorf("request returned error: %s", body)
	}

	var fileInfo struct {
		SHA string `json:"sha"`
	}
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&fileInfo)
	if err != nil {
		return "", false, fmt.Errorf("error decoding request body: %w", err)
	}

	return fileInfo.SHA, true, nil
}

type CommitDataAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CommitData struct {
	Message   string           `json:"message"`
	Content   string           `json:"content"`
	Branch    string           `json:"branch"`
	SHA       string           `json:"sha,omitempty"`
	Committer CommitDataAuthor `json:"committer"`
}

func (gh *GithubRepository) Commit(payload *targets.CommitPayload) error {
	author, email := payload.SplitAuthor()

	for path, file := range payload.Files {
		// Get original file, if exists, for the original file's SHA
		sha, _, err := gh.getFileSHA(path, payload.Branch)
		if err != nil {
			return fmt.Errorf("failed to retrieve file SHA: %w", err)
		}

		b := new(bytes.Buffer)
		err = jsoniter.ConfigFastest.NewEncoder(b).Encode(CommitData{
			Branch:  payload.Branch,
			Message: payload.Message,
			Committer: CommitDataAuthor{
				Name:  author,
				Email: email,
			},
			Content: base64.StdEncoding.EncodeToString(file),
			SHA:     sha,
		})
		if err != nil {
			return fmt.Errorf("failed to encode commit payload: %w", err)
		}

		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/repos/%s/contents/%s", gh.baseURI, gh.projectID, path), b)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/vnd.github.v3+json")
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(gh.credentials)))

		res, err := gh.client.Do(req)
		if err != nil {
			return fmt.Errorf("error performing request: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode >= 400 {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("request returned error: %s", body)
		}
	}

	return nil
}
