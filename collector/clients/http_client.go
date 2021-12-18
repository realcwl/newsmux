package clients

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type HttpClient struct {
	header  http.Header
	cookies []http.Cookie

	client *http.Client
}

func NewDefaultHttpClient() *HttpClient {
	return &HttpClient{header: http.Header{}, cookies: []http.Cookie{}, client: &http.Client{}}
}

func NewHttpClient(header http.Header, cookies []http.Cookie) *HttpClient {
	return &HttpClient{header: header, cookies: cookies, client: &http.Client{}}
}

func NewHttpClientFromTaskParams(task *protocol.PanopticTask) *HttpClient {
	header := http.Header{}
	for _, h := range task.TaskParams.HeaderParams {
		header[h.Key] = []string{h.Value}
	}
	cookies := []http.Cookie{}
	for _, c := range task.TaskParams.Cookies {
		cookies = append(cookies, http.Cookie{Name: c.Key, Value: c.Value})
	}

	return NewHttpClient(header, cookies)
}

func (c *HttpClient) Post(uri string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", uri, body)
	req.Header = c.header
	for _, cookie := range c.cookies {
		req.AddCookie(&cookie)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if IsNon200HttpResponse(res) {
		MaybeLogNon200HttpError(res)
		return nil, errors.New("")
	}

	return res, err
}

func (c *HttpClient) Get(uri string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	req.Header = c.header
	for _, cookie := range c.cookies {
		req.AddCookie(&cookie)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if IsNon200HttpResponse(res) {
		MaybeLogNon200HttpError(res)
		return nil, errors.New("")
	}

	return res, err
}

// This method takes in an additional map from query key to query value, which
// will be appended to query uri as ?${KEY}=${VALUE}
func (c *HttpClient) GetWithQueryParams(uri string, params map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header = c.header
	for k, v := range params {
		req.URL.Query().Add(k, v)
	}
	for _, cookie := range c.cookies {
		req.AddCookie(&cookie)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if IsNon200HttpResponse(res) {
		MaybeLogNon200HttpError(res)
		return nil, errors.New("")
	}

	return res, err
}

// Log http response if the error code is not 2XX
func MaybeLogNon200HttpError(res *http.Response) {
	if IsNon200HttpResponse(res) {
		Logger.Log.Errorf("non-200 http code: %d", res.StatusCode)
		LogHttpResponseBody(res)
	}
}

func IsNon200HttpResponse(res *http.Response) bool {
	return res.StatusCode >= 300
}

func LogHttpResponseBody(res *http.Response) {
	body, err := ioutil.ReadAll(res.Body)
	if err == nil {
		Logger.Log.Errorln("response body is: ", string(body))
	}
}
