package gitea_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	jsoniter "github.com/json-iterator/go"

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

func (ge *GiteaRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/raw/%s?ref=%s", ge.baseURI, ge.projectID, path, url.QueryEscape(ref))
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(ge.credentials)))

	res, err := ge.client.Do(req)
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

func (ge *GiteaRepository) getFileSHA(path, branch string) (string, bool, error) {
	requestURI := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", ge.baseURI, ge.projectID, path, url.QueryEscape(branch))
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return "", false, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(ge.credentials)))

	res, err := ge.client.Do(req)
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
	Message string           `json:"message"`
	Content string           `json:"content"`
	Branch  string           `json:"branch"`
	SHA     string           `json:"sha,omitempty"`
	Author  CommitDataAuthor `json:"author"`
}

func (ge *GiteaRepository) Commit(payload *targets.CommitPayload) error {
	author, email := payload.SplitAuthor()

	for path, file := range payload.Files {
		// Get original file, if exists, for the original file's SHA
		sha, _, err := ge.getFileSHA(path, payload.Branch)
		if err != nil {
			return fmt.Errorf("failed to retrieve file SHA: %w", err)
		}

		b := new(bytes.Buffer)
		err = jsoniter.ConfigFastest.NewEncoder(b).Encode(CommitData{
			Branch:  payload.Branch,
			Message: payload.Message,
			Author: CommitDataAuthor{
				Name:  author,
				Email: email,
			},
			Content: base64.StdEncoding.EncodeToString(file),
			SHA:     sha,
		})
		if err != nil {
			return fmt.Errorf("failed to encode commit payload: %w", err)
		}

		putURI := fmt.Sprintf("%s/repos/%s/contents/%s", ge.baseURI, ge.projectID, path)
		req, err := http.NewRequest("PUT", putURI, b)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(ge.credentials)))

		res, err := ge.client.Do(req)
		if err != nil {
			return fmt.Errorf("error performing request: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode >= 400 {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("request returned error: %s", body)
		}

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
	}

	return nil
}
