package azure_target

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	jsoniter "github.com/json-iterator/go"
	"github.com/neosperience/shipper/targets"
)

// AzureRepository commits to an Azure DevOps Services Git repository using the Azure Git REST APIs
type AzureRepository struct {
	baseURI      string
	projectID    string
	repositoryID string
	credentials  string

	client *http.Client
}

// NewAPIClient creates a AzureRepository instance
func NewAPIClient(projectID string, repositoryID string, credentials string) *AzureRepository {
	var client = &http.Client{}
	return &AzureRepository{
		baseURI:      "https://dev.azure.com/",
		projectID:    projectID,
		repositoryID: repositoryID,
		credentials:  credentials,
		client:       client,
	}
}

func (azure *AzureRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/%s/_apis/git/repositories/%s/items?path=%s&version=%s&api-version=6.0", azure.baseURI, azure.projectID, azure.repositoryID, url.QueryEscape(path), url.QueryEscape(ref))
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(azure.credentials)))

	res, err := azure.client.Do(req)
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

func (azure *AzureRepository) headRef(ref string) (string, error) {
	requestURI := fmt.Sprintf("%s/%s/_apis/git/repositories/%s/refs?filter=heads/%s&$top=1&api-version=6.0", azure.baseURI, azure.projectID, azure.repositoryID, url.QueryEscape(ref))
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(azure.credentials)))

	res, err := azure.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error performing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		return "", fmt.Errorf("request returned error: %s", body)
	}

	var refs refList
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&refs)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	if len(refs.Value) == 0 {
		return "", fmt.Errorf("ref %s not found", ref)
	}

	return refs.Value[0].ObjectID, nil
}

func (azure *AzureRepository) Commit(payload *targets.CommitPayload) error {
	ref, err := azure.headRef(payload.Branch)
	if err != nil {
		return fmt.Errorf("error getting ref: %w", err)
	}

	changes := make([]commitChange, len(payload.Files))
	i := 0
	for path, file := range payload.Files {
		changes[i] = commitChange{
			ChangeType: "edit",
			Item:       pushItem{Path: "/" + strings.TrimLeft(path, "/")},
			NewContent: contentFor(file),
		}
		i += 1
	}

	b := new(bytes.Buffer)
	err = jsoniter.ConfigFastest.NewEncoder(b).Encode(pushData{
		RefUpdates: []pushRef{{
			Name:        "refs/heads/" + payload.Branch,
			OldObjectID: ref,
		}},
		Commits: []commit{{
			Comment: payload.Message,
			Author:  getAuthor(payload),
			Changes: changes,
		}},
	})
	if err != nil {
		return fmt.Errorf("error encoding request payload: %w", err)
	}

	requestURI := fmt.Sprintf("%s/%s/_apis/git/repositories/%s/pushes?api-version=6.0", azure.baseURI, azure.projectID, azure.repositoryID)
	req, err := http.NewRequest("POST", requestURI, b)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(azure.credentials)))
	req.Header.Add("Content-Type", "application/json")

	res, err := azure.client.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("request returned error: %s", body)
	}

	var response pushResponse
	err = jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	if len(response.Commits) == 0 {
		return fmt.Errorf("no commits returned")
	}

	log.Printf("Commit URL: %s/commit/%s", response.Repository.WebURL, response.Commits[0].CommitID)
	return nil
}

// contentFor encodes the file contents either as text or as a base64 string depending on the contents
func contentFor(data []byte) pushContent {
	// if the file is a text file, we can just submit its contents as a string
	if utf8.Valid(data) {
		return pushContent{
			Content:     string(data),
			ContentType: "rawtext",
		}
	}
	return pushContent{
		Content:     base64.StdEncoding.EncodeToString(data),
		ContentType: "base64encoded",
	}
}

func getAuthor(payload *targets.CommitPayload) commitAuthor {
	if payload.Author == "" {
		return commitAuthor{}
	}
	name, email := payload.SplitAuthor()
	return commitAuthor{
		Name:  name,
		Email: email,
		Date:  time.Now(),
	}
}
