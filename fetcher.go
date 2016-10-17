package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
)

type Fetcher interface {
	// Fetch returns the response status of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (status string, urls []string, err error)
}

type HttpFetch struct {
	Client *http.Client
}

func NewHttpFetch() *HttpFetch {
	return &HttpFetch{
		Client: &http.Client{},
	}
}

func (hf *HttpFetch) Fetch(url string) (string, []string, error) {
	res, err := hf.Client.Get(url)
	if err != nil {
		return "", nil, err
	}
	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return res.Status, nil, err
	}
	return res.Status, nil, err
}

// Resolve a potentially relative child URL string against a
// parent URL. Ensure resolved URL does not include a fragment
// portion (ie. the part after a '#')
func resolve(parent *url.URL, child string) (string, error) {
	u, err := parent.Parse(child)
	if err != nil {
		return "", err
	}
	u.Fragment = "" // normalize URLs by dropping the fragment portion after the '#'
	return u.String(), nil
}
