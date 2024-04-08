package req_util

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
)

func MakeHttpRequest(reqMethod string, reqUrl string, reqData url.Values) (*http.Response, []byte, error) {
	req, err := http.NewRequest(reqMethod, reqUrl, bytes.NewBuffer([]byte(reqData.Encode())))
	if err != nil {
		return nil, nil, err
	}
	if len(reqData) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer CloseBody(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, respBody, nil
}

func CloseBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Printf("WARN: Error when closing body: %v", err)
	}
}
