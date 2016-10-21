/*
Package github.com/orirawlings/webcrawler implements a basic HTTP/HTML
webcrawler.

The webcrawler starts at an initial URL, downloads and parses the page for
links, and repeats the process for each subsequent link URL. This
continues up to a maximum specified depth. Downloading and parsing of URLs
is done in separate go routines and no individual URL is crawled more than
once.

As each URL is downloaded a log message is printed to stderr with the HTTP
status code and the URL. If an error occurred during the processing that
is included in the log message as well.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Parse command line arguments. Returns the inital root URL and the
// maximum depth of the crawl.
func parseArgs() (startUrl string, depth int) {
	// Override the default help usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-h] [options] URL\n", os.Args[0])
		flag.PrintDefaults()
	}
	d := flag.Int("depth", 2, "The maximum depth of the web crawl")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Please provide an initial URL to start the crawl")
		os.Exit(1)
	}
	return flag.Arg(0), *d
}

func main() {
	// Set logging time format
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	start, depth := parseArgs()
	done := make(chan struct{}) // Not used. This can be used cancel the crawl early.
	statuses := Crawl(done, start, depth, NewFetch())
	for status := range statuses {
		if status.Err != nil {
			log.Printf("%v\t%v\t%v\n", status.Status, status.Url, status.Err)
		} else {
			log.Printf("%v\t%v\n", status.Status, status.Url)
		}

	}
}
