package collector

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"reflect"

	clients "github.com/Luismorlan/newsmux/collector/clients"
	Logger "github.com/Luismorlan/newsmux/utils/log"
)

func GetCurrentIpAddress(client *clients.HttpClient) (ip string, err error) {
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

	httpClient := clients.HttpClient{}
	httpResponse, err := httpClient.Get(uri)

	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(httpResponse.Body)
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
