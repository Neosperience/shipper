package bitbucket_target

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/neosperience/shipper/common"
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
	var client = &http.Client{}
	return &BitbucketCloudRepository{
		baseURI:     "https://api.bitbucket.org/2.0",
		projectID:   projectID,
		credentials: credentials,
		client:      client,
	}
}

func (bb *BitbucketCloudRepository) doRequest(method string, requestURI string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Add authentication headers
	if headers == nil {
		headers = make(http.Header)
	}
	headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(bb.credentials)))

	return common.HTTPRequest(bb.client, method, requestURI, body, headers)
}

func (bb *BitbucketCloudRepository) Get(path string, ref string) ([]byte, error) {
	requestURI := fmt.Sprintf("%s/repositories/%s/src/%s/%s", bb.baseURI, bb.projectID, ref, path)
	res, err := bb.doRequest("GET", requestURI, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error performing GET file: %w", err)
	}
	defer res.Body.Close()

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

	requestURI := fmt.Sprintf("%s/repositories/%s/src", bb.baseURI, bb.projectID)
	res, err := bb.doRequest("POST", requestURI, strings.NewReader(data.Encode()), http.Header{
		"Content-Type": []string{"application/x-www-form-urlencoded"},
	})
	if err != nil {
		return fmt.Errorf("error performing POST /src: %w", err)
	}
	defer res.Body.Close()

	log.Printf("Commit URL (API): %s", res.Header.Get("Location"))

	return nil
}
