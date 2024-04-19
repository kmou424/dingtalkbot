package dingtalkbot

import (
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/event"
)

type ChatMessage *chatbot.BotCallbackDataModel

type structEventMessage struct {
	Header *event.EventHeader
	Data   *RWMap[string, *Value]
}
type EventMessage *structEventMessage
