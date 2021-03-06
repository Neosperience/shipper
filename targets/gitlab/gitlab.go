package gitlab_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"unicode/utf8"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/common"

	"github.com/neosperience/shipper/targets"
)

// GitlabRepository commits to a GitLab repository using the GitLab REST Commits APIs
type GitlabRepository struct {
	baseURI    string
	projectID  string
	privateKey string

	client *http.Client
}

// NewAPIClient creates a GitlabRepository instance
func NewAPIClient(uri string, projectID string, key string) *GitlabRepository {
	var client = &http.Client{}
	return &GitlabRepository{
		baseURI:    uri,
		projectID:  projectID,
		privateKey: key,
		client:     client,
	}
}

type CommitAction struct {
	Action   string `json:"action"`
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
	Encoding string `json:"encoding"`
}

type CommitPostData struct {
	Branch        string         `json:"branch"`
	CommitMessage string         `json:"commit_message"`
	AuthorName    string         `json:"author_name"`
	AuthorEmail   string         `json:"author_email"`
	Actions       []CommitAction `json:"actions"`
}

type FileInfo struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int    `json:"size"`
	Encoding      string `json:"encoding"`
	Content       string `json:"content"`
	ContentSha256 string `json:"content_sha256"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
}

func (gl *GitlabRepository) doRequest(method string, requestURI string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Add authentication headers
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Set("PRIVATE-TOKEN", gl.privateKey)

	return common.HTTPRequest(gl.client, method, requestURI, body, headers)
}

func (gl *GitlabRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/projects/%s/repository/files/%s?ref=%s", gl.baseURI, url.PathEscape(gl.projectID), url.PathEscape(path), url.QueryEscape(ref))
	res, err := gl.doRequest("GET", requestURI, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error retrieving file from GitLab: %w", err)
	}
	defer res.Body.Close()

	var fileinfo FileInfo
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&fileinfo)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	switch fileinfo.Encoding {
	case "text":
		return []byte(fileinfo.Content), nil
	case "base64":
		return base64.StdEncoding.DecodeString(fileinfo.Content)
	default:
		return nil, fmt.Errorf("received file in unsupported encoding: %s", fileinfo.Encoding)
	}
}

func (gl *GitlabRepository) Commit(payload *targets.CommitPayload) error {
	actions := []CommitAction{}
	for name, content := range payload.Files {
		// Encode as text or base64 depending on wheter content is a valid string
		if utf8.Valid(content) {
			actions = append(actions, CommitAction{
				Action:   "update",
				FilePath: name,
				Encoding: "text",
				Content:  string(content),
			})
		} else {
			actions = append(actions, CommitAction{
				Action:   "update",
				FilePath: name,
				Encoding: "base64",
				Content:  base64.StdEncoding.EncodeToString(content),
			})
		}
	}

	author, email := payload.SplitAuthor()

	b := new(bytes.Buffer)
	err := jsoniter.ConfigFastest.NewEncoder(b).Encode(CommitPostData{
		Branch:        payload.Branch,
		CommitMessage: payload.Message,
		AuthorName:    author,
		AuthorEmail:   email,
		Actions:       actions,
	})
	if err != nil {
		return fmt.Errorf("error encoding request payload: %w", err)
	}

	requestURI := fmt.Sprintf("%s/projects/%s/repository/commits", gl.baseURI, url.PathEscape(gl.projectID))
	res, err := gl.doRequest("POST", requestURI, b, http.Header{
		"Content-Type": []string{"application/json"},
	})
	if err != nil {
		return fmt.Errorf("error pushing commit to GitLab API: %w", err)
	}
	defer res.Body.Close()

	var response struct {
		WebURL string `json:"web_url"`
	}
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	log.Printf("Commit URL: %s", response.WebURL)

	return nil
}
