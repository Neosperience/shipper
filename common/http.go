package common

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func HTTPRequest(client *http.Client, method string, requestURI string, body io.Reader, headers http.Header) (*http.Response, error) {
	// Create request
	req, err := http.NewRequest(method, requestURI, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	// Add headers and authorization
	for k, v := range headers {
		req.Header[k] = v
	}

	// Perform request and check for errors
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}
	if res.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(res.Body)
		_ = res.Body.Close()
		return res, fmt.Errorf("request returned error: %s", body)
	}
	return res, nil
}
