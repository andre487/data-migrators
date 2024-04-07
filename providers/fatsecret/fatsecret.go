package fatsecret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/andre487/data-migrators/utils/req_util"
)

const authUrl = "https://oauth.fatsecret.com/connect/token"
const apiUrl = "https://platform.fatsecret.com/rest/server.api"

type FatSecret struct {
	creds credentials
}

type credentials struct {
	ClientId        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	Token           string
	TokenBorderTime int64
}

type tokenData struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func (c *credentials) RequestToken() error {
	authBody := []byte("grant_type=client_credentials&scope=basic")
	req, err := http.NewRequest("POST", authUrl, bytes.NewBuffer(authBody))
	if err != nil {
		return fmt.Errorf("error when creating FatSecret token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.ClientId, c.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error when requesting token: %v", err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("WARN: Error when closing body: %v", err)
		}
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error when requesting token: %v", err)
	}

	if resp.StatusCode > 201 {
		return fmt.Errorf("error when requesting token, HTTP %d: %v", resp.StatusCode, string(respBody))
	}

	var bodyData tokenData
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		return fmt.Errorf("error when requesting token, invalid body: %v", err)
	}

	if bodyData.TokenType != "Bearer" {
		return fmt.Errorf("error when requesting token, invalid token type: %v", bodyData.TokenType)
	}

	if bodyData.Scope != "basic" {
		return fmt.Errorf("error when requesting token, invalid scope: %v", bodyData.Scope)
	}

	c.Token = bodyData.AccessToken
	c.TokenBorderTime = time.Now().Unix() + bodyData.ExpiresIn

	return nil
}

func (c *credentials) RequestTokenIfExpired() error {
	if time.Now().Unix() >= c.TokenBorderTime {
		return c.RequestToken()
	}
	return nil
}

func New(keyData []byte, options ...func(s *FatSecret)) (*FatSecret, error) {
	creds := credentials{}
	if err := json.Unmarshal(keyData, &creds); err != nil {
		return nil, fmt.Errorf("invalid credentials format: %v", err)
	}
	if err := creds.RequestToken(); err != nil {
		return nil, err
	}

	p := &FatSecret{creds: creds}
	for _, opt := range options {
		opt(p)
	}
	return p, nil
}

func (s *FatSecret) makeApiRequest(method string, reqData map[string]string) (map[string]interface{}, error) {
	if err := s.creds.RequestTokenIfExpired(); err != nil {
		return nil, err
	}

	reqBodyParams := url.Values{"method": []string{method}, "format": []string{"json"}}
	for key, value := range reqData {
		reqBodyParams.Set(key, value)
	}
	reqBody := bytes.NewBuffer([]byte(reqBodyParams.Encode()))

	req, err := http.NewRequest("POST", apiUrl, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error when creating FatSecret API request: method=%s, err=%v", method, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.creds.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error when making API request: method=%s, err=%v", method, err)
	}
	defer req_util.CloseBody(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error when retrieving API result: method=%s, err=%v", method, err)
	}

	if resp.StatusCode > 201 {
		return nil, fmt.Errorf("error when making API request, HTTP %d: method=%s, %v", resp.StatusCode, method, string(respBody))
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(respBody, &bodyData); err != nil {
		return nil, fmt.Errorf("error when requesting token, invalid body: %v", err)
	}

	apiErr := bodyData["error"].(map[string]interface{})
	if apiErr != nil {
		errCode := uint64(apiErr["code"].(float64))
		errMsg := apiErr["message"].(string)
		return nil, fmt.Errorf("API error: method=%s, code=%d: %s", method, errCode, errMsg)
	}

	return bodyData, nil
}

func (s *FatSecret) GetTestData() (interface{}, error) {
	return s.makeApiRequest("food.get.v2", map[string]string{
		"food_id": "33691",
	})
}
