package dingtalkbot

func (m *Messenger) Send(msg Sendable) {
	m.enqueueMessage(msg)
}

func (m *Messenger) SendTextMessage(conversationId, text string) {
	msg := &DingTalkMessage{
		MsgKey: "sampleText",
		MsgParam: map[string]string{
			"content": text,
		},
		extras:         m.requireParams("robotCode"),
		ConversationId: conversationId,
	}
	m.Send(msg)
}

func (m *Messenger) SendMarkdownMessage(conversationId, title, text string) {
	msg := &DingTalkMessage{
		MsgKey: "sampleMarkdown",
		MsgParam: map[string]string{
			"title": title,
			"text":  text,
		},
		extras:         m.requireParams("robotCode"),
		ConversationId: conversationId,
	}
	m.Send(msg)
}
