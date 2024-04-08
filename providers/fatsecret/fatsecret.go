package fatsecret

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
)

const apiUrl = "https://platform.fatsecret.com/rest/server.api"

type FatSecret struct {
	oauth *FatSOauth1Service
}

func New(keyData []byte, options ...func(s *FatSecret)) (*FatSecret, error) {
	keys := FatSOauth1Keys{}
	if err := json.Unmarshal(keyData, &keys); err != nil {
		return nil, fmt.Errorf("error when parsing FatSecret keys: %v", err)
	}
	oauth := NewFatSOauth1Service(keys)
	err := oauth.Authorize()
	if err != nil {
		log.Fatal(err)
	}

	p := &FatSecret{oauth: oauth}
	for _, opt := range options {
		opt(p)
	}
	return p, nil
}

func (s *FatSecret) makeApiRequest(method string, reqData map[string]string) (map[string]interface{}, error) {
	reqBodyParams := url.Values{"method": []string{method}, "format": []string{"json"}}
	for key, value := range reqData {
		reqBodyParams.Set(key, value)
	}
	resp, respBody, err := s.oauth.MakeHttpRequest("POST", apiUrl, reqBodyParams)
	if err != nil {
		return nil, fmt.Errorf("error when making request for method %s: %v", method, err)
	}

	if resp.StatusCode > 201 {
		return nil, fmt.Errorf("error when making API request, HTTP %d: method=%s, %v", resp.StatusCode, method, string(respBody))
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		return nil, fmt.Errorf("error when requesting token, invalid body: %v", err)
	}

	if bodyData["error"] != nil {
		apiErr := bodyData["error"].(map[string]interface{})
		errCode := uint64(apiErr["code"].(float64))
		errMsg := apiErr["message"].(string)
		return nil, fmt.Errorf("API error: method=%s, code=%d: %s", method, errCode, errMsg)
	}

	return bodyData, nil
}

func (s *FatSecret) GetTestData() (interface{}, error) {
	return s.makeApiRequest("food_entries.get.v2", map[string]string{})
}
