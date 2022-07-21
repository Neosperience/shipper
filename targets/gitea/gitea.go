package gitea_target

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

// GiteaRepository commits to a Gitea repository using the Gitea REST Commits APIs
type GiteaRepository struct {
	baseURI     string
	projectID   string
	credentials string

	client *http.Client
}

// NewAPIClient creates a GiteaRepository instance
func NewAPIClient(uri string, projectID string, credentials string) *GiteaRepository {
	var client = &http.Client{}
	return &GiteaRepository{
		baseURI:     uri,
		projectID:   projectID,
		credentials: credentials,
		client:      client,
	}
}

func (ge *GiteaRepository) doRequest(method string, requestURI string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Add authentication headers
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(ge.credentials)))

	return common.HTTPRequest(ge.client, method, requestURI, body, headers)
}

func (ge *GiteaRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/raw/%s?ref=%s", ge.baseURI, ge.projectID, path, url.QueryEscape(ref))
	res, err := ge.doRequest("GET", requestURI, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting file: %w", err)
	}
	defer res.Body.Close()

	return ioutil.ReadAll(res.Body)
}

func (ge *GiteaRepository) getFileSHA(path, branch string) (string, bool, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", ge.baseURI, ge.projectID, path, url.QueryEscape(branch))
	res, err := ge.doRequest("GET", requestURI, nil, nil)
	if err != nil && !isNotFound(res) {
		return "", false, fmt.Errorf("error getting file SHA from server: %w", err)
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
	Message string           `json:"message"`
	Content string           `json:"content"`
	Branch  string           `json:"branch"`
	SHA     string           `json:"sha,omitempty"`
	Author  CommitDataAuthor `json:"author"`
}

func (ge *GiteaRepository) commitSingle(path string, commitData CommitData) error {
	// Get original file, if exists, for the original file's SHA
	sha, _, err := ge.getFileSHA(path, commitData.Branch)
	if err != nil {
		return fmt.Errorf("failed to retrieve file SHA: %w", err)
	}
	commitData.SHA = sha

	b := new(bytes.Buffer)
	err = jsoniter.ConfigFastest.NewEncoder(b).Encode(commitData)
	if err != nil {
		return fmt.Errorf("failed to encode commit payload: %w", err)
	}

	putURI := fmt.Sprintf("%s/repos/%s/contents/%s", ge.baseURI, ge.projectID, path)
	res, err := ge.doRequest("PUT", putURI, b, http.Header{
		"Content-Type": []string{"application/json"},
	})
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
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

func (ge *GiteaRepository) Commit(payload *targets.CommitPayload) error {
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
		err := ge.commitSingle(path, CommitData{
			Branch:  payload.Branch,
			Message: message,
			Author:  commitAuthor,
			Content: base64.StdEncoding.EncodeToString(file),
		})
		if err != nil {
			return fmt.Errorf("error committing file %s: %w", path, err)
		}
	}

	return nil
}

func isNotFound(res *http.Response) bool {
	return res != nil && res.StatusCode == 404
}
