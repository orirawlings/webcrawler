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

// Crawl uses fetcher to crawl pages starting with url, to a maximum of depth.
// Urls will be crawled at most once. To preemptively terminate crawling,
// close channel 'done'.
func Crawl(done <-chan struct{}, urlStr string, depth int, fetcher Fetcher) <-chan *CrawlStatus {
	var wg sync.WaitGroup
	seen := newSeenSet()
	out := make(chan *CrawlStatus)

	var crawl func(string, int)
	crawl = func(urlStr string, depth int) {
		defer wg.Done()
		if depth <= 0 || fetcher == nil {
			return
		}
		status, urls, err := fetcher.Fetch(urlStr)
		cs := &CrawlStatus{urlStr, status, err}
		select {
		case out <- cs:
		case <-done:
			return
		}
		if err == nil {
			for _, u := range urls {
				if ok := seen.ensure(u); !ok {
					wg.Add(1)
					go crawl(u, depth-1)
				}
			}
		}
	}

	// Crawl the initial url
	seen.ensure(urlStr)
	wg.Add(1)
	go crawl(urlStr, depth)

	// Close the output channel once all crawling has terminated
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Synchronized set to track which urls have already
// been scheduled for crawling
type seenSet struct {
	mux  sync.Mutex
	urls map[string]struct{}
}

func newSeenSet() *seenSet {
	return &seenSet{urls: make(map[string]struct{})}
}

// Ensure url is added to seenSet and return true if it
// was already present, false otherwise
func (s *seenSet) ensure(url string) bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	if _, ok := s.urls[url]; ok {
		return true
	}
	s.urls[url] = struct{}{}
	return false
}
