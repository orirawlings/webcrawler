package main

import (
	"log"
	"net/url"
	"sync"
)

type Fetcher interface {
	// Fetch returns the response status of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (status string, urls []string, err error)
}

type FetchStatus struct {
	// The url that was fetched
	Url string
	// The HTTP status code recieved when fetching Url
	Status string
	// The error produced when fetching the Url
	Err error
}

// Resolve a potentially relative child URL string against a
// parent URL. Ensure resolved URL does not include a fragment
// portion (ie. the part after a '#')
func normalize(parent *url.URL, child string) (string, error) {
	u, err := parent.Parse(child)
	if err != nil {
		return "", err
	}
	u.Fragment = "" // normalize URLs by dropping the fragment portion after the '#'
	return u.String(), nil
}

// Crawl uses fetcher to crawl pages starting
// with url, to a maximum of depth. To preemptively
// terminate crawling, close channel 'done'.
func Crawl(done <-chan struct{}, urlStr string, depth int, fetcher Fetcher) <-chan *FetchStatus {
	var wg sync.WaitGroup
	var mux sync.Mutex
	seen := make(map[string]bool)
	out := make(chan *FetchStatus)

	var crawl func(string, int)
	crawl = func(urlStr string, depth int) {
		defer wg.Done()
		if depth <= 0 || fetcher == nil {
			return
		}
		status, urls, err := fetcher.Fetch(urlStr)
		fs := &FetchStatus{
			Status: status,
			Url:    urlStr,
			Err:    err,
		}
		select {
		case out <- fs:
		case <-done:
			return
		}
		if err != nil {
			return
		}
		parent, err := url.ParseRequestURI(urlStr)
		if err != nil {
			return
		}
		mux.Lock()
		defer mux.Unlock()
		for _, u := range urls {
			u, err = normalize(parent, u)
			if err != nil {
				continue
			}
			if _, ok := seen[u]; !ok {
				seen[u] = true
				wg.Add(1)
				go crawl(u, depth-1)
			}
		}
	}

	// Crawl the initial url
	mux.Lock()
	seen[urlStr] = true
	mux.Unlock()
	wg.Add(1)
	go crawl(urlStr, depth)

	// Close the output channel once all crawling has terminated
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	done := make(chan struct{})
	statuses := Crawl(done, "http://golang.org/", 4, NewHttpFetch())
	for status := range statuses {
		log.Printf("%v\t%v\t%v\n", status.Url, status.Status, status.Err)
	}
}
