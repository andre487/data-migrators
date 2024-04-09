package req_util

import (
	"bytes"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"time"
)

var retryStatuses = map[int]bool{
	429: true,
	500: true,
	501: true,
	502: true,
	503: true,
	504: true,
}

type HttpRequestRetryConfig struct {
	Retries     int
	Backoff     time.Duration
	MaxTimeout  time.Duration
	RetryNumber int
}

func MakeHttpRequest(reqMethod string, reqUrl string, reqData url.Values, retryConfig *HttpRequestRetryConfig) (*http.Response, []byte, error) {
	if retryConfig == nil {
		retryConfig = &HttpRequestRetryConfig{
			Retries:    5,
			Backoff:    time.Second / 10,
			MaxTimeout: time.Second * 60,
		}
	}

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

	if retryConfig.Retries > 0 && retryStatuses[resp.StatusCode] == true && retryConfig.RetryNumber < retryConfig.Retries {
		waitTime := retryConfig.Backoff * time.Duration(math.Pow(2, float64(retryConfig.RetryNumber)))
		retryConfig.RetryNumber++
		log.Printf("WARN: Retry HTTP request because of status %d, retryNumber=%d, waitTime=%s", resp.StatusCode, retryConfig.RetryNumber, waitTime.String())
		time.Sleep(waitTime)
		return MakeHttpRequest(reqMethod, reqUrl, reqData, retryConfig)
	}

	return resp, respBody, nil
}

func CloseBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Printf("WARN: Error when closing body: %v", err)
	}
}
