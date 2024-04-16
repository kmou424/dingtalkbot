package dingtalkbot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/charmbracelet/log"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/event"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/payload"
	. "github.com/open-dingtalk/dingtalk-stream-sdk-go/utils"
)

type ChatMessage *BotCallbackDataModel

type structEventMessage struct {
	Header *EventHeader
	Data   *RWMap[string, *Value]
}
type EventMessage *structEventMessage

type Bot struct {
	*Messenger

	clientId     string
	clientSecret string

	c *StreamClient

	chatHandlers  []HandlerFunc[ChatMessage]
	eventHandlers []HandlerFunc[EventMessage]

	cancel context.CancelFunc
}

func NewBot(clientId, clientSecret string) (*Bot, error) {
	bot := &Bot{
		clientId:     clientId,
		clientSecret: clientSecret,
	}
	messenger, err := newMessenger()
	if err != nil {
		return nil, err
	}
	bot.initStorage(messenger.storage)
	bot.Messenger = messenger

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

func (bot *Bot) initStorage(storage map[string]string) {
	storage["clientId"] = bot.clientId
	storage["clientSecret"] = bot.clientSecret
	storage["robotCode"] = bot.clientId
}

func (bot *Bot) AutoReconnect() *Bot {
	WithAutoReconnect(true)(bot.c)
	return bot
}

func (bot *Bot) SetDebug(debug bool) *Bot {
	if debug {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
	return bot
}

func (bot *Bot) onChatReceived(ctx context.Context, data *BotCallbackDataModel) (_ []byte, err error) {
	select {
	case <-ctx.Done():
		return
	default:
		err = (&Context[ChatMessage]{
			Message:  data,
			Bot:      bot,
			handlers: bot.chatHandlers,
		}).start()
	}
	return
}

func (bot *Bot) onEventReceived(ctx context.Context, header *EventHeader, rawData []byte) (_ EventProcessStatusType, err error) {
	select {
	case <-ctx.Done():
		return EventProcessStatusKSuccess, nil
	default:
		data := newRWValueMap[string]()
		err = json.Unmarshal(rawData, data)
		if err != nil {
			return EventProcessStatusKLater, err
		}
		err = (&Context[EventMessage]{
			Message: &structEventMessage{
				Header: header,
				Data:   data,
			},
			Bot:      bot,
			handlers: bot.eventHandlers,
		}).start()
		if err != nil {
			return EventProcessStatusKLater, err
		}
	}
	return EventProcessStatusKSuccess, nil
}

func (bot *Bot) HandleChat(handler ChatHandlerFunc) *Bot {
	bot.chatHandlers = append(bot.chatHandlers, HandlerFunc[ChatMessage](handler))
	return bot
}

func (bot *Bot) HandleEvent(handler EventHandlerFunc) *Bot {
	bot.eventHandlers = append(bot.eventHandlers, HandlerFunc[EventMessage](handler))
	return bot
}

func (bot *Bot) Start() error {
	SetLogger(&iLogger{})

	ctx, cancelFunc := context.WithCancel(context.Background())
	bot.cancel = cancelFunc

	bot.Messenger.start(ctx)

	err := bot.c.Start(ctx)
	if err != nil {
		return err
	}
	defer bot.c.Close()

	<-ctx.Done()

	return nil
}

func (bot *Bot) Stop() error {
	if bot.cancel == nil {
		return errors.New("can't stop a never started bot")
	}

	bot.cancel()
	return nil
}
