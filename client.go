package dingtalkbot

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dgraph-io/badger/v4"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	dingClient "github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/event"
	dingLogger "github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/payload"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/utils"
	"github.com/zyedidia/generic/queue"
)

type Client struct {
	*Messenger

	clientId     string
	clientSecret string

	dClient *dingClient.StreamClient

	modules *RWMap[MessageType, Module]

	cache     *badger.DB

	cancel    context.CancelFunc
	destroyed bool
}

func NewClient(id, secret string) (client *Client, err error) {
	client = (&Client{
		clientId:     id,
		clientSecret: secret,
		modules:      newRWMap[MessageType, Module](),
	}).Debug(false)

	err = client.initCache()
	if err != nil {
		return nil, err
	}

	// init messenger
	client.Messenger = &Messenger{
		cache:       client.cache,
		mqm:         newRWMap[string, *queue.Queue[Sendable]](),
		mq:          make(chan Sendable, 10),
		storage:     make(map[string]string),
		tokenExpiry: time.Now(),
	}
	func(storage map[string]string) {
		storage["clientId"] = id
		storage["clientSecret"] = secret
		storage["robotCode"] = id
	}(client.Messenger.storage)

	client.dClient = dingClient.NewStreamClient(
		dingClient.WithAppCredential(
			dingClient.NewAppCredentialConfig(id, secret),
		),
		dingClient.WithUserAgent(
			dingClient.NewDingtalkGoSDKUserAgent(),
		),
		dingClient.WithSubscription(
			utils.SubscriptionTypeKCallback,
			payload.BotMessageCallbackTopic,
			chatbot.NewDefaultChatBotFrameHandler(client.onChatReceived).OnEventReceived,
		),
		dingClient.WithSubscription(
			utils.SubscriptionTypeKEvent,
			"*",
			event.NewDefaultEventFrameHandler(client.onEventReceived).OnEventReceived,
		),
	)
	return
}

func (client *Client) initCache() (err error) {
	options := badger.DefaultOptions("").WithInMemory(true)
	client.cache, err = badger.Open(options)
	return
}

func (c *Client) onMessage(message *Message) error {
	module, ok := c.modules.Get(message.Type)
	// if message type never registered
	if !ok {
		return nil
	}
	middlewares, handler := module.handlers(message)
	return (&Context{
		Message:     message,
		Client:      c,
		middlewares: middlewares,
		handler:     handler,
	}).handling()
}

func (c *Client) onEventReceived(ctx context.Context, header *event.EventHeader, rawData []byte) (_ event.EventProcessStatusType, err error) {
	select {
	case <-ctx.Done():
		return event.EventProcessStatusKSuccess, nil
	default:
		data := newRWValueMap[string]()
		err = json.Unmarshal(rawData, data)
		if err != nil {
			return event.EventProcessStatusKLater, err
		}
		eventMsg := new(struct {
			Header *event.EventHeader
			data   *RWMap[string, *Value]
		})
		eventMsg.Header = header
		eventMsg.data = data
		message := toMessage(EventMessage(eventMsg))
		err = c.onMessage(message)
		if err != nil {
			return event.EventProcessStatusKLater, err
		}
	}
	return event.EventProcessStatusKSuccess, nil
}

func (c *Client) onChatReceived(ctx context.Context, data *chatbot.BotCallbackDataModel) (_ []byte, err error) {
	select {
	case <-ctx.Done():
		return
	default:
		message := toMessage(ChatMessage(data))
		err = c.onMessage(message)
	}
	return
}

func (c *Client) AutoReconnect() *Client {
	dingClient.WithAutoReconnect(true)(c.dClient)
	return c
}

func (c *Client) Debug(debug bool) *Client {
	if debug {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
	return c
}

func (c *Client) Start() error {
	if c.destroyed {
		return errors.New("bot has been destroyed")
	}
	dingLogger.SetLogger(&iLogger{})

	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancel = cancelFunc

	c.Messenger.start(ctx)

	err := c.dClient.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	defer func() {
		c.dClient.Close()
		close(c.Messenger.mq)
		c.destroyed = true
	}()

	return nil
}

func (c *Client) Stop() error {
	if c.cancel == nil {
		return errors.New("can't stop a never started bot")
	}

	c.cancel()
	return nil
}

func (c *Client) Register(messageType MessageType, module Module) *Client {
	c.modules.Put(messageType, module)
	return c
}