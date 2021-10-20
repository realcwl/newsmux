package collector

import (
	"net/http"
	"time"
)

type HttpClient struct {
	header  http.Header
	cookies []http.Cookie
}

func NewHttpClient(header http.Header, cookies []http.Cookie) *HttpClient {
	return &HttpClient{header: header, cookies: cookies}
}

func (c HttpClient) Get(uri string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	req.Header = c.header
	for _, cookie := range c.cookies {
		req.AddCookie(&cookie)
	}
	return client.Do(req)
}

func (HttpClient) GetWithin(uri string, seconds int) (resp *http.Response, err error) {
	client := &http.Client{Timeout: time.Duration(seconds) * time.Second}
	return client.Get(uri)
}
