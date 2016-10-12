package main

import (
	"fmt"
	"testing"
)

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string][]string

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if urls, ok := f[url]; ok {
		return "200", urls, nil
	}
	return "404", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"http://golang.org/": []string{
		"http://golang.org/pkg/",
		"http://golang.org/cmd/",
	},
	"http://golang.org/pkg/": []string{
		"http://golang.org/",
		"http://golang.org/cmd/",
		"http://golang.org/pkg/fmt/",
		"http://golang.org/pkg/os/",
	},
	"http://golang.org/pkg/fmt/": []string{
		"http://golang.org/",
		"http://golang.org/pkg/",
	},
	"http://golang.org/pkg/os/": []string{
		"http://golang.org/",
		"http://golang.org/pkg/",
	},
}

func TestFail(t *testing.T) {
	t.Error("This test fails")
}
