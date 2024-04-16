package dingtalkbot

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"
	"reflect"
)

type Sendable interface {
	json.Marshaler
	OpenConversationId() string
}

type DingTalkMessage struct {
	MsgKey         string            `json:"msgKey" mapstructure:"msgKey"`
	MsgParam       map[string]string `json:"msgParam" mapstructure:"msgParam"`
	ConversationId string            `json:"openConversationId" mapstructure:"openConversationId"`

	extras map[string]string
}

//goland:noinspection GoMixedReceiverTypes
func (msg *DingTalkMessage) OpenConversationId() string {
	return msg.ConversationId
}

//goland:noinspection GoMixedReceiverTypes
func (msg DingTalkMessage) MarshalJSON() ([]byte, error) {
	dst := make(map[string]any)
	err := mapstructure.Decode(msg, &dst)
	if err != nil {
		return nil, err
	}
	dstExtras := make(map[string]any)
	err = mapstructure.Decode(msg.extras, &dstExtras)
	if err != nil {
		return nil, err
	}
	for _, key := range maps.Keys(dstExtras) {
		dst[key] = dstExtras[key]
	}

	if msgParam, ok := dst["msgParam"]; ok && reflect.TypeOf(msgParam).Kind() != reflect.String {
		msgParamBytes, err := json.Marshal(msgParam)
		if err != nil {
			return nil, err
		}
		dst["msgParam"] = string(msgParamBytes)
	}

	return json.Marshal(dst)
}
