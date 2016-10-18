package main

import (
	"errors"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Fetcher interface {
	// Fetch returns the response status of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (status string, urls []string, err error)
}

type Client interface {
	Get(url string) (r *http.Response, err error)
}

type HttpFetch struct {
	Client Client
}

func NewHttpFetch() *HttpFetch {
	return &HttpFetch{
		Client: &http.Client{},
	}
}

func (hf *HttpFetch) Fetch(urlStr string) (string, []string, error) {
	res, err := hf.Client.Get(urlStr)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()
	parent, err := url.Parse(urlStr)
	if err != nil {
		return res.Status, nil, err
	}
	urls, err := ParseLinks(res.Body, parent)
	return res.Status, urls, err
}

func atAnchorTag(z *html.Tokenizer) bool {
	tag, hasAttr := z.TagName()
	return string(tag) == "a" && hasAttr
}

var HrefNotFound = errors.New("href attribute not found for <a> tag")

func resolveAnchorHref(z *html.Tokenizer, parent *url.URL) (string, error) {
	key, value, more := z.TagAttr()
	for {
		if string(key) == "href" {
			href := strings.TrimSpace(string(value))
			return resolve(parent, href)
		}
		if !more {
			break
		}
		key, value, more = z.TagAttr()
	}
	return "", HrefNotFound
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

// Parse HTML response and return all the urls linked from this
// response within <a> tags
func ParseLinks(r io.Reader, parent *url.URL) ([]string, error) {
	z := html.NewTokenizer(r)
	result := make([]string, 0)
	for {
		t := z.Next()
		switch t {
		case html.ErrorToken:
			if err := z.Err(); err != io.EOF {
				return result, err
			}
			return result, nil
		case html.StartTagToken, html.SelfClosingTagToken:
			if atAnchorTag(z) {
				url, err := resolveAnchorHref(z, parent)
				if err == nil {
					result = append(result, url)
				}
			}
		}
	}
}
