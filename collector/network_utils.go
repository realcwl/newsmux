package collector

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/Luismorlan/newsmux/protocol"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

type HttpClient struct {
	header  http.Header
	cookies []http.Cookie
}

func NewHttpClient(header http.Header, cookies []http.Cookie) *HttpClient {
	return &HttpClient{header: header, cookies: cookies}
}

func (c HttpClient) Get(uri string) (*http.Response, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	req.Header = c.header
	for _, cookie := range c.cookies {
		req.AddCookie(&cookie)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if IsNon200HttpResponse(res) {
		MaybeLogNon200HttpError(res)
		return nil, errors.New("")
	}

	return res, err
}

func GetCurrentIpAddress(client HttpClient) (ip string, err error) {
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	resp.Body.Close()
	return string(body), err
}

func (HttpClient) GetWithin(uri string, seconds int) (resp *http.Response, err error) {
	client := &http.Client{Timeout: time.Duration(seconds) * time.Second}
	return client.Get(uri)
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
	body, err := io.ReadAll(res.Body)
	if err == nil {
		Logger.Log.Errorln("response body is: ", string(body))
	}
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

// HttpGetAndParseResponse will make an HTTP request to the specified URI. Then,
// it will parse the body as JSON into the specified response. Return error on
// any failure.
// Note that, failure not only include network issue, any non 200 response code
// will also be considered as a failure.
// The response passed in must be a pointer.
func HttpGetAndParseJsonResponse(uri string, res interface{}) error {
	if reflect.ValueOf(res).Type().Kind() != reflect.Ptr {
		return errors.New("the passed in variable must be a pointer")
	}

	httpClient := HttpClient{}
	httpResponse, err := httpClient.Get(uri)

	if err != nil {
		return err
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	// Remove BOM before parsing, see https://en.wikipedia.org/wiki/Byte_order_mark for details.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	err = json.Unmarshal(body, res)
	if err != nil {
		Logger.Log.Errorf("fail to parse response: %s, type: %T", body, res)
		return err
	}

	return nil
}
