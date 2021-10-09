package collector

import (
	"net/http"
)

type HttpClient struct{}

func (HttpClient) Get(uri string) (resp *http.Response, err error) {
	return http.Get(uri)
}
