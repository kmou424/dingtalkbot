package dingtalkbot

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/event"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/payload"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/utils"
	"golang.org/x/exp/maps"
)

type ChatMessage *BotCallbackDataModel

type structEventMessage struct {
	Header *EventHeader
	Data   *RWMap[string, *Value]
}
type EventMessage *structEventMessage

type IBot interface {
	Messenger() *Messenger
	chatHandlerEntry(ChatMessage) error
	eventHandlerEntry(EventMessage) error
	Start() error
	Stop() error
}

type BaseBot struct {
	messenger *Messenger

	clientId     string
	clientSecret string

	c *StreamClient

	chatHandler  HandlerFunc[ChatMessage]
	eventHandler HandlerFunc[EventMessage]

	cancel    context.CancelFunc
	destroyed bool
}

func NewBaseBot(clientId, clientSecret string) (*BaseBot, error) {
	bot := &BaseBot{
		clientId:     clientId,
		clientSecret: clientSecret,
	}
	var err error
	bot.messenger, err = newMessenger()
	if err != nil {
		return nil, err
	}

	func(storage map[string]string) {
		storage["clientId"] = bot.clientId
		storage["clientSecret"] = bot.clientSecret
		storage["robotCode"] = bot.clientId
	}(bot.messenger.storage)

	bot.c = NewStreamClient(
		WithAppCredential(NewAppCredentialConfig(clientId, clientSecret)),
		WithUserAgent(NewDingtalkGoSDKUserAgent()),
		WithSubscription(
			SubscriptionTypeKCallback,
			BotMessageCallbackTopic,
			NewDefaultChatBotFrameHandler(bot.onChatReceived).OnEventReceived,
		),
		WithSubscription(
			SubscriptionTypeKEvent,
			"*",
			NewDefaultEventFrameHandler(bot.onEventReceived).OnEventReceived,
		),
	)

	return bot, nil
}

func (bot *BaseBot) AutoReconnect() *BaseBot {
	WithAutoReconnect(true)(bot.c)
	return bot
}

func (bot *BaseBot) SetDebug(debug bool) *BaseBot {
	if debug {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
	return bot
}

func (bot *BaseBot) Messenger() *Messenger {
	return bot.messenger
}

func (bot *BaseBot) onChatReceived(ctx context.Context, data *BotCallbackDataModel) (_ []byte, err error) {
	select {
	case <-ctx.Done():
		return
	default:
		err = bot.chatHandlerEntry(data)
	}
	return
}

func (bot *BaseBot) onEventReceived(ctx context.Context, header *EventHeader, rawData []byte) (_ EventProcessStatusType, err error) {
	select {
	case <-ctx.Done():
		return EventProcessStatusKSuccess, nil
	default:
		data := newRWValueMap[string]()
		err = json.Unmarshal(rawData, data)
		if err != nil {
			return EventProcessStatusKLater, err
		}
		message := &structEventMessage{
			Header: header,
			Data:   data,
		}
		err = bot.eventHandlerEntry(message)
		if err != nil {
			return EventProcessStatusKLater, err
		}
	}
	return EventProcessStatusKSuccess, nil
}

func (bot *BaseBot) chatHandlerEntry(msg ChatMessage) error {
	return (&Context[ChatMessage]{
		Message: msg,
		Bot:     bot,
		handler: bot.chatHandler,
	}).handling()
}

func (bot *BaseBot) eventHandlerEntry(msg EventMessage) error {
	return (&Context[EventMessage]{
		Message: msg,
		Bot:     bot,
		handler: bot.eventHandler,
	}).handling()
}

func (bot *BaseBot) HandleChat(handler ChatHandlerFunc) *BaseBot {
	bot.chatHandler = HandlerFunc[ChatMessage](handler)
	return bot
}

func (bot *BaseBot) HandleEvent(handler EventHandlerFunc) *BaseBot {
	bot.eventHandler = HandlerFunc[EventMessage](handler)
	return bot
}

func (bot *BaseBot) Start() error {
	if bot.destroyed {
		return errors.New("bot has been destroyed")
	}
	SetLogger(&iLogger{})

	ctx, cancelFunc := context.WithCancel(context.Background())
	bot.cancel = cancelFunc

	bot.messenger.start(ctx)

	err := bot.c.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	defer func() {
		bot.c.Close()
		close(bot.messenger.mq)
		bot.destroyed = true
	}()

	return nil
}

func (bot *BaseBot) Stop() error {
	if bot.cancel == nil {
		return errors.New("can't stop a never started bot")
	}

	bot.cancel()
	return nil
}

// MiddlewareBot handler for command like "/cmd", every command will hit all middlewares and one handler
type MiddlewareBot struct {
	base                 *BaseBot
	chatMiddlewares      []HandlerFunc[ChatMessage]
	eventMiddlewares     []HandlerFunc[EventMessage]
	chatHandlersMapping  map[string]HandlerFunc[ChatMessage]
	eventHandlersMapping map[string]HandlerFunc[EventMessage]
}

const (
	defaultChatCommand = "/*"
	defaultEventKey    = "*"

	commandPrefix = "/"
	eventPrefix   = "event:"
)

func (bot *BaseBot) MiddlewareBot() *MiddlewareBot {
	return &MiddlewareBot{
		base:                 bot,
		chatMiddlewares:      []HandlerFunc[ChatMessage]{},
		eventMiddlewares:     []HandlerFunc[EventMessage]{},
		chatHandlersMapping:  map[string]HandlerFunc[ChatMessage]{},
		eventHandlersMapping: map[string]HandlerFunc[EventMessage]{},
	}
}

func (bot *MiddlewareBot) Messenger() *Messenger {
	return bot.base.Messenger()
}

func (bot *MiddlewareBot) chatHandlerEntry(msg ChatMessage) error {
	msgContent := strings.Trim(msg.Text.Content, " ")
	if !strings.HasPrefix(msgContent, "/") {
		return nil
	}

	targetCmd := defaultChatCommand
	validCmdList := maps.Keys(bot.chatHandlersMapping)
	sort.Strings(validCmdList)
	for _, command := range validCmdList {
		if strings.HasPrefix(msgContent, command) {
			targetCmd = command
		}
	}

	// can't hit valid handler of command
	if !slices.Contains(validCmdList, targetCmd) {
		return nil
	}

	return (&Context[ChatMessage]{
		Message:     msg,
		Bot:         bot,
		middlewares: bot.chatMiddlewares,
		handler:     bot.chatHandlersMapping[targetCmd],
	}).handling()
}

func (bot *MiddlewareBot) eventHandlerEntry(msg EventMessage) error {
	receivedKey := eventPrefix + msg.Header.EventType

	targetEvent := defaultEventKey
	validEventList := maps.Keys(bot.eventHandlersMapping)
	sort.Strings(validEventList)
	for _, event := range validEventList {
		if receivedKey == event {
			targetEvent = event
		}
	}

	// can't hit valid handler of event
	if !slices.Contains(validEventList, targetEvent) {
		return nil
	}

	return (&Context[EventMessage]{
		Message:     msg,
		Bot:         bot,
		middlewares: bot.eventMiddlewares,
		handler:     bot.eventHandlersMapping[targetEvent],
	}).handling()
}

func (bot *MiddlewareBot) Start() error {
	return bot.base.Start()
}

func (bot *MiddlewareBot) Stop() error {
	return bot.base.Stop()
}
