package github_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/common"

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

func (gh *GithubRepository) doRequest(method string, requestURI string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Add authentication headers
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(gh.credentials)))

	return common.HTTPRequest(gh.client, method, requestURI, body, headers)
}

func (gh *GithubRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", gh.baseURI, gh.projectID, path, url.QueryEscape(ref))
	res, err := gh.doRequest("GET", requestURI, nil, http.Header{
		"Accept": {"application/vnd.github.v3.raw"},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting file: %w", err)
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func (gh *GithubRepository) getFileSHA(path, branch string) (string, bool, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", gh.baseURI, gh.projectID, path, url.QueryEscape(branch))
	res, err := gh.doRequest("GET", requestURI, nil, http.Header{
		"Accept": {"application/vnd.github.v3+json"},
	})
	if err != nil && !isNotFound(res) {
		return "", false, fmt.Errorf("error getting file SHA: %w", err)
	}

	// File not found, file does not exist
	if res.StatusCode == 404 {
		return "", false, nil
	}

	defer res.Body.Close()

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

func (gh *GithubRepository) commitSingle(path string, commitData CommitData) error {
	// Get original file, if exists, for the original file's SHA
	sha, _, err := gh.getFileSHA(path, commitData.Branch)
	if err != nil {
		return fmt.Errorf("failed to retrieve file SHA: %w", err)
	}
	commitData.SHA = sha

	b := new(bytes.Buffer)
	err = jsoniter.ConfigFastest.NewEncoder(b).Encode(commitData)
	if err != nil {
		return fmt.Errorf("failed to encode commit payload: %w", err)
	}

	requestURI := fmt.Sprintf("%s/repos/%s/contents/%s", gh.baseURI, gh.projectID, path)
	res, err := gh.doRequest("PUT", requestURI, b, http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/vnd.github.v3+json"},
	})
	if err != nil {
		return fmt.Errorf("error pushing file to GitHub API: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		Commit struct {
			HTMLURL string `json:"html_url"`
		} `json:"commit"`
	}
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return fmt.Errorf("error decoding response body: %w", err)
	}

	log.Printf("Commit URL: %s", response.Commit.HTMLURL)
	return nil
}

func (gh *GithubRepository) Commit(payload *targets.CommitPayload) error {
	author, email := payload.SplitAuthor()
	commitAuthor := CommitDataAuthor{
		Name:  author,
		Email: email,
	}

	multipleFiles := len(payload.Files) > 1
	for path, file := range payload.Files {
		message := payload.Message
		if multipleFiles {
			message = fmt.Sprintf("%s: %s", payload.Message, path)
		}

		err := gh.commitSingle(path, CommitData{
			Branch:    payload.Branch,
			Message:   message,
			Committer: commitAuthor,
			Content:   base64.StdEncoding.EncodeToString(file),
		})
		if err != nil {
			return fmt.Errorf("failed to commit file %s: %w", path, err)
		}
	}

	return nil
}

func isNotFound(res *http.Response) bool {
	return res != nil && res.StatusCode == 404
}
