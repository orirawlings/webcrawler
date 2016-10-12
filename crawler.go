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
	Url, Status string
	Err         error
}

// Crawl uses fetcher to crawl pages starting
// with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) <-chan FetchStatus {
	seen, mux, wg := make(map[string]bool), sync.Mutex{}, sync.WaitGroup{}
	out := make(chan FetchStatus)
	var crawl func(url string, depth int)
	crawl = func(url string, depth int) {
		defer wg.Done()
		if depth <= 0 {
			return
		}
		status, urls, err := fetcher.Fetch(url)
		fs := FetchStatus{
			Status: status,
			Url:    url,
			Err:    err,
		}
		out <- fs
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
	statuses := Crawl("http://golang.org/", 4, nil)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	for status := range statuses {
		log.Printf("%v\t%v\t%v\n", status.Url, status.Status, status.Err)
	}
}
