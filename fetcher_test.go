package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const (
	OK = "200 OK"
	NF = "404 Not Found"
)

// fakeClient is a client that returns canned results.
type fakeClient map[string]*http.Response

var fakeClientErr = errors.New("Could not find response")

// Return the canned result or error if it does not exist
func (c *fakeClient) Get(url string) (r *http.Response, err error) {
	if r, ok := (*c)[url]; ok {
		return r, nil
	}
	return nil, fakeClientErr
}

var client = &fakeClient{
	"http://golang.org/about/someone": &http.Response{
		Status: OK,
		Body: withLinks([]string{
			"http://golang.org/pkg/",
			"http://golang.org/cmd/",
			"/blog/",       // Resolve relative to root
			"me/",          // Resolve relative to current path
			" \t/~user/\n", // Trim whitespace
			"you#top",      // Drop fragment
		}),
	},
	"http://example.org/empty": &http.Response{
		Status: OK,
		Body:   withLinks(make([]string, 0)),
	},
	"http://example.org/notFound": &http.Response{
		Status: NF,
		Body:   ioutil.NopCloser(strings.NewReader("")),
	},
}

var expected = map[string]struct {
	status string
	urls   []string
	err    error
}{
	"http://golang.org/about/someone": {
		status: OK,
		urls: []string{
			"http://golang.org/pkg/",
			"http://golang.org/cmd/",
			"http://golang.org/blog/",     // Resolve relative to root
			"http://golang.org/about/me/", // Resolve relative to current path
			"http://golang.org/~user/",    // Trim whitespace
			"http://golang.org/about/you", // Drop fragment
		},
	},
	"http://example.org/empty": {
		status: OK,
		urls:   make([]string, 0, 0),
	},
	"http://example.org/notFound": {
		status: NF,
		urls:   make([]string, 0, 0),
	},
	"https://example.org/notStubbed": {
		err: fakeClientErr,
	},
}

func TestFetch(t *testing.T) {
	f := &HttpFetch{Client: client}
	for url, exp := range expected {
		s, us, err := f.Fetch(url)
		if s != exp.status {
			t.Errorf("Expected status [%v], was [%v] when fetching url [%s]", exp.status, s, url)
		}
		if !reflect.DeepEqual(us, exp.urls) {
			t.Errorf("Expected child urls [%s], was [%s] when fetching url [%s]", exp.urls, us, url)
		}
		if err != exp.err {
			t.Errorf("Expected err [%v], was [%v] when fetching url [%s]", exp.err, err, url)
		}
	}
}

func withLinks(urls []string) io.ReadCloser {
	b := "<html><body>"
	for _, url := range urls {
		b = b + fmt.Sprintf("<a href=\"%s\">link</a>", url)
	}
	b = b + "</body></html>"
	return ioutil.NopCloser(strings.NewReader(b))
}
