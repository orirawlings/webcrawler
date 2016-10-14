package main

import (
	"log"
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

// Crawl uses fetcher to crawl pages starting
// with url, to a maximum of depth.
func Crawl(done <-chan struct{}, url string, depth int, fetcher Fetcher) <-chan FetchStatus {
	var wg sync.WaitGroup
	var mux sync.Mutex
	seen := make(map[string]bool)
	out := make(chan FetchStatus)

	var crawl func(url string, depth int)
	crawl = func(url string, depth int) {
		defer wg.Done()
		if depth <= 0 || fetcher == nil {
			return
		}
		status, urls, err := fetcher.Fetch(url)
		fs := FetchStatus{
			Status: status,
			Url:    url,
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
		mux.Lock()
		defer mux.Unlock()
		for _, u := range urls {
			if _, ok := seen[u]; !ok {
				seen[u] = true
				wg.Add(1)
				go crawl(u, depth-1)
			}
		}
	}

	// Crawl the initial url
	mux.Lock()
	seen[url] = true
	mux.Unlock()
	wg.Add(1)
	go crawl(url, depth)

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
	statuses := Crawl(done, "http://golang.org/", 4, nil)
	for status := range statuses {
		log.Printf("%v\t%v\t%v\n", status.Url, status.Status, status.Err)
	}
}
