package fatsecret

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andre487/data-migrators/utils/req_util"
	"github.com/andre487/data-migrators/utils/storage"
)

const requestTokenUrl = "https://www.fatsecret.com/oauth/request_token"
const accessTokenUrl = "https://www.fatsecret.com/oauth/access_token"
const authorizeUrl = "https://www.fatsecret.com/oauth/authorize"

var digitsRe, _ = regexp.Compile("^\\d+$")

type FatSOauth1Service struct {
	Keys FatSOauth1Keys

	storage  *storage.Storage
	authData struct {
		AuthCode           string
		RequestToken       string
		RequestTokenSecret string
		AccessToken        string
		AccessTokenSecret  string
	}
}

type FatSOauth1Keys struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

type fatSSecretData struct {
	Value  string `json:"value"`
	Value2 string `json:"value2"`
	Time   uint64 `json:"time"`
}

func NewFatSOauth1Service(keys FatSOauth1Keys) *FatSOauth1Service {
	return &FatSOauth1Service{
		Keys:    keys,
		storage: storage.New("fatsecret_oauth"),
	}
}

func (s *FatSOauth1Service) MakeHttpRequest(reqMethod string, reqUrl string, reqData url.Values) (*http.Response, []byte, error) {
	reqData, err := s.addOauthParams(reqMethod, reqUrl, reqData)
	if err != nil {
		return nil, nil, fmt.Errorf("error when creating OAuth params for request: %v", err)
	}

	resp, respBody, err := req_util.MakeHttpRequest(reqMethod, reqUrl, reqData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error when making OAuth signed request: %v", err)
	}
	return resp, respBody, nil
}

func (s *FatSOauth1Service) Authorize() error {
	if err := s.GetRequestToken(); err != nil {
		return err
	}

	cacheName := "access_token"
	cachedData := s.getCachedSecret(cacheName)
	if cachedData != nil && cachedData.Value != "" && cachedData.Value2 != "" {
		s.authData.AccessToken = cachedData.Value
		s.authData.AccessTokenSecret = cachedData.Value2
		return nil
	}

	if err := s.GetAuthCode(); err != nil {
		return err
	}

	if s.authData.AccessToken != "" && s.authData.AccessTokenSecret != "" {
		return nil
	}

	reqData := url.Values{
		"oauth_token":    []string{s.authData.RequestToken},
		"oauth_verifier": []string{s.authData.AuthCode},
	}
	oauthParams, err := s.addOauthParams("POST", accessTokenUrl, reqData)
	if err != nil {
		return fmt.Errorf("OAuth request token params error: %v", err)
	}

	resp, respBody, err := req_util.MakeHttpRequest("POST", accessTokenUrl, oauthParams, nil)
	if err != nil {
		return fmt.Errorf("OAuth request token creation error: %v", err)
	}

	if resp.StatusCode > 201 {
		return fmt.Errorf("OAuth token request HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	resultParams, err := url.ParseQuery(string(respBody))
	if err != nil {
		return fmt.Errorf("error when parsing token response: %v", err)
	}

	if val := resultParams.Get("oauth_token"); val != "" {
		s.authData.AccessToken = val
	} else {
		return fmt.Errorf("OAuth response: there is no oauth_token in response")
	}

	if val := resultParams.Get("oauth_token_secret"); val != "" {
		s.authData.AccessTokenSecret = val
	} else {
		return fmt.Errorf("OAuth response: there is no oauth_token_secret in response")
	}

	s.setCachedSecret(cacheName, s.authData.AccessToken, s.authData.AccessTokenSecret)
	return nil
}

func (s *FatSOauth1Service) GetAuthCode() error {
	cacheName := "auth_code"
	cachedData := s.getCachedSecret(cacheName)
	if cachedData != nil && cachedData.Value != "" {
		s.authData.AuthCode = cachedData.Value
		return nil
	}

	if err := s.GetRequestToken(); err != nil {
		return err
	}

	fmt.Println("==> Go to the authorize URL and enter code")
	fmt.Printf("Authorize URL: %s\n", authorizeUrl+"?oauth_token="+s.authData.RequestToken)

	fmt.Print("Enter code: ")
	reader := bufio.NewReader(os.Stdin)
	val, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error when reading authorize token: %v", err)
	}
	val = strings.TrimSpace(val)

	if !digitsRe.Match([]byte(val)) {
		return fmt.Errorf("invalid authorization code: %s", val)
	}
	s.authData.AuthCode = val
	s.setCachedSecret(cacheName, val, "")

	return nil
}

func (s *FatSOauth1Service) GetRequestToken() error {
	cacheName := "request_token"
	cachedData := s.getCachedSecret(cacheName)
	if cachedData != nil && cachedData.Value != "" && cachedData.Value2 != "" {
		s.authData.RequestToken = cachedData.Value
		s.authData.RequestTokenSecret = cachedData.Value2
		return nil
	}

	if s.authData.RequestToken != "" && s.authData.RequestTokenSecret != "" {
		return nil
	}

	reqData := url.Values{"oauth_callback": []string{"oob"}}
	oauthParams, err := s.addOauthParams("POST", requestTokenUrl, reqData)
	if err != nil {
		return fmt.Errorf("OAuth request token params error: %v", err)
	}

	resp, respBody, err := req_util.MakeHttpRequest("POST", requestTokenUrl, oauthParams, nil)
	if err != nil {
		return fmt.Errorf("OAuth request token creation error: %v", err)
	}

	if resp.StatusCode > 201 {
		return fmt.Errorf("OAuth token request HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	resultParams, err := url.ParseQuery(string(respBody))
	if err != nil {
		return fmt.Errorf("error when parsing token response: %v", err)
	}

	if val := resultParams.Get("oauth_callback_confirmed"); val != "true" {
		return fmt.Errorf("OAuth callback not confirmed: %s != true", val)
	}

	if val := resultParams.Get("oauth_token"); val != "" {
		s.authData.RequestToken = val
	} else {
		return fmt.Errorf("OAuth response: there is no oauth_token in response")
	}

	if val := resultParams.Get("oauth_token_secret"); val != "" {
		s.authData.RequestTokenSecret = val
	} else {
		return fmt.Errorf("OAuth response: there is no oauth_token_secret in response")
	}

	s.setCachedSecret(cacheName, s.authData.RequestToken, s.authData.RequestTokenSecret)
	return nil
}

func (s *FatSOauth1Service) getCachedSecret(name string) *fatSSecretData {
	fileName := fmt.Sprintf("fatsecret_oauth_%s.json", name)
	filePath := s.storage.GetFile(fileName, 0600)

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("error reading FatSOauth1Service secret file: %v", err)
	}

	res := fatSSecretData{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil
	}
	return &res
}

func (s *FatSOauth1Service) setCachedSecret(name string, value string, value2 string) {
	fileName := fmt.Sprintf("fatsecret_oauth_%s.json", name)
	filePath := s.storage.GetFile(fileName, 0600)

	data := fatSSecretData{
		Value:  value,
		Value2: value2,
		Time:   uint64(time.Now().Unix()),
	}

	var res []byte
	var err error
	if res, err = json.Marshal(data); err != nil {
		log.Fatalf("error when serializing secret data: %v", err)
	}

	if err := os.WriteFile(filePath, res, 0600); err != nil {
		log.Fatalf("error when writing secret data: %v", err)
	}
}

func (s *FatSOauth1Service) addOauthParams(reqMethod string, reqUrl string, reqData url.Values) (url.Values, error) {
	nonceBuilder := strings.Builder{}
	for i := 0; i < 8; i++ {
		nonceBuilder.WriteString(strconv.FormatInt(rand.Int64()%10, 10))
	}
	nonce := nonceBuilder.String()

	vals := url.Values{}
	for name, val := range reqData {
		vals.Set(name, val[0])
	}

	vals.Set("oauth_consumer_key", url.QueryEscape(s.Keys.ConsumerKey))
	vals.Set("oauth_nonce", nonce)
	vals.Set("oauth_signature_method", "HMAC-SHA1")
	vals.Set("oauth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	if s.authData.AccessToken != "" {
		vals.Set("oauth_token", url.QueryEscape(s.authData.AccessToken))
	}
	vals.Set("oauth_version", "1.0")

	signature, err := s.signParams(reqMethod, reqUrl, vals)
	if err != nil {
		return nil, err
	}
	vals.Set("oauth_signature", signature)

	return vals, nil
}

func (s *FatSOauth1Service) signParams(reqMethod string, reqUrl string, params url.Values) (string, error) {
	urlData, err := url.Parse(reqUrl)
	if err != nil {
		return "", err
	}
	cleanUrl := fmt.Sprintf("%s://%s%s", urlData.Scheme, urlData.Host, urlData.Path)

	baseString := fmt.Sprintf("%s&%s&%s", reqMethod, url.QueryEscape(cleanUrl), url.QueryEscape(params.Encode()))
	baseString = strings.Replace(baseString, "+", "%20", -1)
	baseString = strings.Replace(baseString, "%7E", "~", -1)

	key := url.QueryEscape(s.Keys.ConsumerSecret) + "&"
	oauthTokenParam := params.Get("oauth_token")
	if oauthTokenParam != "" {
		if oauthTokenParam == url.QueryEscape(s.authData.AccessToken) {
			key += s.authData.AccessTokenSecret
		} else if oauthTokenParam == url.QueryEscape(s.authData.RequestToken) {
			key += s.authData.RequestTokenSecret
		}
	}

	digest := hmac.New(sha1.New, []byte(key))
	digest.Write([]byte(baseString))
	return base64.StdEncoding.EncodeToString(digest.Sum(nil)), nil
}
