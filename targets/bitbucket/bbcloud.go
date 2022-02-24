package bitbucket_target

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neosperience/shipper/targets"
)

// BitbucketCloudRepository commits to a Bitbucket.com repository using the Bitbucket Cloud REST APIs
type BitbucketCloudRepository struct {
	baseURI     string
	projectID   string
	credentials string

	client *http.Client
}

// NewCloudAPIClient creates a BitbucketCloudRepository instance
func NewCloudAPIClient(projectID string, credentials string) *BitbucketCloudRepository {
	var transport = &http.Transport{
		IdleConnTimeout: 30 * time.Second,
	}
	var client = &http.Client{Transport: transport}
	return &BitbucketCloudRepository{
		baseURI:     "https://api.bitbucket.org/2.0",
		projectID:   projectID,
		credentials: credentials,
		client:      client,
	}
}

func (bb *BitbucketCloudRepository) Get(path string, ref string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repositories/%s/src/%s/%s", bb.baseURI, bb.projectID, ref, path), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(bb.credentials)))

	res, err := bb.client.Do(req)
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

func (bb *BitbucketCloudRepository) Commit(payload *targets.CommitPayload) error {
	data := url.Values{}
	for name, content := range payload.Files {
		trailName := "/" + strings.TrimLeft(name, "/")
		data.Set(trailName, string(content))
	}
	data.Set("branch", payload.Branch)
	data.Set("author", payload.Author)
	data.Set("message", payload.Message)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/repositories/%s/src", bb.baseURI, bb.projectID), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(bb.credentials)))

	res, err := bb.client.Do(req)
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
