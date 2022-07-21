package common

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neosperience/shipper/test"
)

func TestHTTPRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello World"))
	}))
	defer server.Close()

	client := server.Client()

	res, err := HTTPRequest(client, "GET", server.URL, nil, nil)
	test.MustSucceed(t, err, "Failed to perform request")
	test.AssertExpected(t, res.StatusCode, http.StatusOK, "Request status code doesn't match expected value")

	// Read body
	body, err := ioutil.ReadAll(res.Body)
	test.MustSucceed(t, err, "Failed to read response body")
	test.AssertExpected(t, string(body), "Hello World", "Request body doesn't match expected value")
}

func TestHTTPRequestWithHeaders(t *testing.T) {
	headerName := "Test-Header"
	headerValue := "Test-Value"

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(headerName) != headerValue {
			t.Fatalf("Request header %s doesn't match expected value", headerName)
		}
	}))
	defer server.Close()

	client := server.Client()

	_, err := HTTPRequest(client, "GET", server.URL, nil, http.Header{
		headerName: []string{headerValue},
	})
	test.MustSucceed(t, err, "Failed to perform request")
}

func TestHTTPRequestWithBody(t *testing.T) {
	testBody := "Hello World"

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		test.MustSucceed(t, err, "Failed to read request body")
		test.AssertExpected(t, string(body), testBody, "Request body doesn't match expected value")
	}))

	client := server.Client()

	_, err := HTTPRequest(client, "GET", server.URL, strings.NewReader(testBody), nil)
	test.MustSucceed(t, err, "Failed to perform request")
}

func TestFaultyServer(t *testing.T) {
	// Mock server that just errors out
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := server.Client()
	_, err := HTTPRequest(client, "GET", server.URL, nil, nil)
	test.MustFail(t, err, "Request supposed to error out for error response but call exited successfully")

	// Try requesting an unreachable server
	_, err = HTTPRequest(client, "GET", "http://localhost:1/invalid", nil, nil)
	test.MustFail(t, err, "Request supposed to error out for unreachable server but call exited successfully")

	// Try requesting an invalid values
	_, err = HTTPRequest(client, "😐", "invalid@@", nil, nil)
	test.MustFail(t, err, "Request supposed to error out for invalid method/URI but call exited successfully")
}
