package dingtalkbot

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"time"
)

const baseUrl = "https://api.dingtalk.com"
const baseTimeout = 5 * time.Second

type reqHeader struct {
	AccessToken string `json:"x-acs-dingtalk-access-token" mapstructure:"x-acs-dingtalk-access-token"`
}

var client = resty.New().
	SetBaseURL(baseUrl).
	SetContentLength(true).
	SetTimeout(baseTimeout).
	OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
		header, _ := json.Marshal(request.Header)
		body, _ := json.Marshal(request.Body)
		logger.Debug(
			"REQUESTING",
			"url", request.URL,
			"method", request.Method,
			"headers", string(header),
			"body", string(body),
		)
		return nil
	})

func request(headers *reqHeader) *resty.Request {
	req := client.R()
	if headers != nil {
		headersMap := make(map[string]string)
		err := mapstructure.Decode(headers, &headersMap)
		if err != nil {
			logger.Warn("convert headers failed", "err", err)
			return req
		}
		req.SetHeaders(headersMap)
	}
	return req
}

func post(path string, body map[string]any, headers *reqHeader) ([]byte, error) {
	resp, err := request(headers).
		SetBody(body).
		SetHeader("Content-Type", "application/json").
		Post(path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("response status code: %d, body=%v", resp.StatusCode(), string(resp.Body()))
	}
	return resp.Body(), nil
}
