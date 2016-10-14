package main

import (
	"io/ioutil"
	"net/http"
)

type HttpFetch struct {
	Client *http.Client
}

func NewHttpFetch() *HttpFetch {
	return &HttpFetch{
		Client: &http.Client{},
	}
}

func (hf *HttpFetch) Fetch(url string) (string, []string, error) {
	res, err := hf.Client.Get(url)
	if err != nil {
		return "", nil, err
	}
	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return res.Status, nil, err
	}
	return res.Status, nil, err
}
