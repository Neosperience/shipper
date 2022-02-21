package gitlab_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
	"unicode/utf8"

	jsoniter "github.com/json-iterator/go"
	"gitlab.neosperience.com/tools/shipper/targets"
)

// GitlabTarget commits to a GitLab repository using the GitLab REST Commits APIs
type GitlabTarget struct {
	baseURI    string
	projectID  string
	privateKey string

	client *http.Client
}

// NewTarget initialized a GitlabTarget instance
func NewTarget(uri string, projectID string, key string) *GitlabTarget {
	var transport = &http.Transport{
		IdleConnTimeout: 30 * time.Second,
	}
	var client = &http.Client{Transport: transport}
	return &GitlabTarget{
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

func (gl *GitlabTarget) Commit(payload targets.CommitPayload) error {
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

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/projects/%s/repository/commits", gl.baseURI, url.PathEscape(gl.projectID)), b)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", gl.privateKey)

	res, err := gl.client.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("request returned error: %s", body)
	}

	return nil
}
