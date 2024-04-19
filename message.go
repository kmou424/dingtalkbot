package dingtalkbot

import (
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/event"
)

type MessageType string

const (
	TypeChat  MessageType = "Chat"
	TypeEvent MessageType = "Event"
)

type (
	ChatMessage  *chatbot.BotCallbackDataModel
	EventMessage *(struct {
		Header *event.EventHeader
		data   *RWMap[string, *Value]
	})
)

type Message struct {
	Type MessageType
	data any
}

func (m *Message) Event() *EventMessage {
	return m.data.(*EventMessage)
}

func (m *Message) Chat() *ChatMessage {
	return m.data.(*ChatMessage)
}

func toMessage(data any) *Message {
	return &Message{
		data: data,
		Type: func() MessageType {
			switch data.(type) {
			case ChatMessage:
				return TypeChat
			case EventMessage:
				return TypeEvent
			}
			return "unknown"
		}(),
	}
}
