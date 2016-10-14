package main

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string][]string

var ErrNotFound = errors.New("not found")

func (f *fakeFetcher) Fetch(url string) (string, []string, error) {
	if urls, ok := (*f)[url]; ok {
		return "200", urls, nil
	}
	return "404", nil, ErrNotFound
}

// fetcher is a populated fakeFetcher.
var fetcher = &fakeFetcher{
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
		"/about#team",
		"string#value/s",
	},
	"http://golang.org/pkg/os/": []string{
		"http://golang.org/",
		"http://golang.org/pkg/",
		"http://golang.org/pkg/#header",
		"/help",
		"darwin",
	},
}

// The set of urls we expect to find when initiating a crawl at http://golang.org/ to various depths
var expectedUrls = []map[string]struct{}{
	make(map[string]struct{}),
	map[string]struct{}{
		"http://golang.org/": struct{}{},
	},
	map[string]struct{}{
		"http://golang.org/":     struct{}{},
		"http://golang.org/pkg/": struct{}{},
		"http://golang.org/cmd/": struct{}{},
	},
	map[string]struct{}{
		"http://golang.org/":         struct{}{},
		"http://golang.org/pkg/":     struct{}{},
		"http://golang.org/cmd/":     struct{}{},
		"http://golang.org/pkg/fmt/": struct{}{},
		"http://golang.org/pkg/os/":  struct{}{},
	},
	map[string]struct{}{
		"http://golang.org/":               struct{}{},
		"http://golang.org/pkg/":           struct{}{},
		"http://golang.org/cmd/":           struct{}{},
		"http://golang.org/pkg/fmt/":       struct{}{},
		"http://golang.org/pkg/fmt/string": struct{}{},
		"http://golang.org/about":          struct{}{},
		"http://golang.org/pkg/os/":        struct{}{},
		"http://golang.org/pkg/os/darwin":  struct{}{},
		"http://golang.org/help":           struct{}{},
	},
	map[string]struct{}{
		"http://golang.org/":               struct{}{},
		"http://golang.org/pkg/":           struct{}{},
		"http://golang.org/cmd/":           struct{}{},
		"http://golang.org/pkg/fmt/":       struct{}{},
		"http://golang.org/pkg/fmt/string": struct{}{},
		"http://golang.org/about":          struct{}{},
		"http://golang.org/pkg/os/":        struct{}{},
		"http://golang.org/pkg/os/darwin":  struct{}{},
		"http://golang.org/help":           struct{}{},
	},
}

func TestCrawlDoesNotEmitDuplicateUrls(t *testing.T) {
	d := make(chan struct{})
	for i := range expectedUrls {
		seen := make(map[string]struct{})
		fs := Crawl(d, "http://golang.org/", i, fetcher)
		for f := range fs {
			if _, ok := seen[f.Url]; ok {
				t.Errorf("Saw [%v] more than once during crawl of depth %d", f.Url, i)
			}
			seen[f.Url] = struct{}{}
		}
	}
}

func TestCrawlFindsMoreUrlsAsDepthIncreases(t *testing.T) {
	d := make(chan struct{})
	for i, urls := range expectedUrls {
		fs := Crawl(d, "http://golang.org/", i, fetcher)
		for f := range fs {
			if _, ok := urls[f.Url]; !ok {
				t.Errorf("Saw unexpected url [%v] during crawl of depth %d", f.Url, i)
			}
			s, _, err := fetcher.Fetch(f.Url)
			if s != f.Status {
				t.Errorf("Expected Status [%v] for url [%v], saw [%v]", s, f.Url, f.Status)
			}
			if err != f.Err {
				t.Errorf("Expected Err [%v] for url [%v], saw [%v]", err, f.Url, f.Err)
			}
		}
	}
}

type trackingFetcher struct {
	mux    sync.Mutex
	counts map[string]int
	f      *fakeFetcher
}

func NewTrackingFetcher(f *fakeFetcher) *trackingFetcher {
	return &trackingFetcher{
		counts: make(map[string]int),
		f:      f,
	}
}

func (t *trackingFetcher) Fetch(url string) (string, []string, error) {
	t.mux.Lock()
	t.counts[url] = t.counts[url] + 1
	t.mux.Unlock()
	return t.f.Fetch(url)
}

func TestCrawlDoesNotFetchDuplicateUrls(t *testing.T) {
	d := make(chan struct{})
	for i := range expectedUrls {
		tf := NewTrackingFetcher(fetcher)
		fs := Crawl(d, "http://golang.org/", i, tf)
		for _ = range fs {
			// consume the channel until close to allow all fetching to complete
		}
		for url, c := range tf.counts {
			if c > 1 {
				t.Errorf("Fetched [%v] more than once ([%d] times) during crawl of depth %d", url, c, i)
			}
		}
	}
}

func TestCrawlNilFetcher(t *testing.T) {
	d := make(chan struct{})
	for i := range expectedUrls {
		fs := Crawl(d, "http://golang.org/", i, nil)
		for f := range fs {
			t.Errorf("Unexpected FetchStatus for url [%v] received when using nil Fetcher at crawl depth [%d]", f.Url, i)
		}
	}
}

func CheckForLeakedGoroutines(t *testing.T, initialNum int) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		num := runtime.NumGoroutine()
		if num <= initialNum {
			return
		}
		if now := time.Now(); now.After(deadline) || now.Equal(deadline) {
			t.Errorf("Goroutines were leaked, [%d] more goroutines than expected", num-initialNum)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestCrawlWithEarlyPreemptiveTermination(t *testing.T) {
	initial := runtime.NumGoroutine()
	for i := range expectedUrls {
		d := make(chan struct{})
		fs := Crawl(d, "http://golang.org/", i, fetcher)
		_ = <-fs // Consume a single status, but leave others unconsumed
		close(d) // Signal to Crawl that we are giving up on waiting for more FetchStatus
	}
	CheckForLeakedGoroutines(t, initial)
}
