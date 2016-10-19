package main

import "sync"

// Emitted status for each URL that is crawled
type CrawlStatus struct {
	// The url that was crawled
	Url string
	// The HTTP status code recieved when crawling Url
	Status string
	// The error produced when crawling the Url
	Err error
}

// Crawl uses fetcher to crawl pages starting
// with url, to a maximum of depth. To preemptively
// terminate crawling, close channel 'done'.
func Crawl(done <-chan struct{}, urlStr string, depth int, fetcher Fetcher) <-chan *CrawlStatus {
	var wg sync.WaitGroup
	var mux sync.Mutex
	seen := make(map[string]struct{})
	out := make(chan *CrawlStatus)

	var crawl func(string, int)
	crawl = func(urlStr string, depth int) {
		defer wg.Done()
		if depth <= 0 || fetcher == nil {
			return
		}
		status, urls, err := fetcher.Fetch(urlStr)
		fs := &CrawlStatus{
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
		mux.Lock()
		defer mux.Unlock()
		for _, u := range urls {
			if _, ok := seen[u]; !ok {
				seen[u] = struct{}{}
				wg.Add(1)
				go crawl(u, depth-1)
			}
		}
	}

	// Crawl the initial url
	mux.Lock()
	seen[urlStr] = struct{}{}
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
