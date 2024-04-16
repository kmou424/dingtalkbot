package dingtalkbot

import (
	"encoding/json"
)

var (
	openApiGetAccessToken = "/v1.0/oauth2/accessToken"
	openApiSendMessage    = "/v1.0/robot/groupMessages/send"
)

func getAccessToken(clientId, clientSecret string) (accessToken string, expireInSec int, err error) {
	body := map[string]any{
		"appKey":    clientId,
		"appSecret": clientSecret,
	}
	respBody, err := post(openApiGetAccessToken, body, nil)
	if err != nil {
		return
	}
	respBodyMap := make(map[string]any)
	err = json.Unmarshal(respBody, &respBodyMap)
	if err != nil {
		return
	}
	accessToken = respBodyMap["accessToken"].(string)
	expireInSec = int(respBodyMap["expireIn"].(float64))
	return
}

func sendMessage(accessToken string, msg Sendable) (processQueryKey string, err error) {
	body := make(map[string]any)
	// TODO: use more performance method
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(jsonBytes, &body)
	if err != nil {
		return
	}
	headers := &reqHeader{
		AccessToken: accessToken,
	}
	respBody, err := post(openApiSendMessage, body, headers)
	if err != nil {
		return
	}
	respBodyMap := make(map[string]string)
	err = json.Unmarshal(respBody, &respBodyMap)
	if err != nil {
		return
	}
	processQueryKey = respBodyMap["processQueryKey"]
	return
}
