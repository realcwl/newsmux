package collector

import (
	"net/http"
	"time"
)

type HttpClient struct{}

func (HttpClient) Get(uri string) (resp *http.Response, err error) {
	return http.Get(uri)
}

func (HttpClient) GetWithin(uri string, seconds int) (resp *http.Response, err error) {
	client := &http.Client{Timeout: time.Duration(seconds) * time.Second}
	return client.Get(uri)
}
