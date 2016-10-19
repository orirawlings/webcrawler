package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func parseArgs() (startUrl string, depth int) {
	d := flag.Int("depth", 2, "The maximum depth of the breadth first web crawl")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Please provide an initial URL to start the crawl")
		os.Exit(1)
	}
	return flag.Arg(0), *d
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	start, depth := parseArgs()
	done := make(chan struct{})
	statuses := Crawl(done, start, depth, NewFetch())
	for status := range statuses {
		if status.Err != nil {
			log.Printf("%v\t%v\t%v\n", status.Status, status.Url, status.Err)
		} else {
			log.Printf("%v\t%v\n", status.Status, status.Url)
		}

	}
}
