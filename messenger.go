package dingtalkbot

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/zyedidia/generic/queue"
	"time"
)

const MQScanInterval = time.Second
const CachePrefix = "message_%s_"

type Messenger struct {
	cache *badger.DB

	mqm     *RWMap[string, *queue.Queue[Sendable]]
	mq      chan Sendable
	storage map[string]string

	accessToken string
	tokenExpiry time.Time
}

func newMessenger() (*Messenger, error) {
	options := badger.DefaultOptions("").WithInMemory(true)
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}
	return &Messenger{
		cache:       db,
		mqm:         newRWMap[string, *queue.Queue[Sendable]](),
		mq:          make(chan Sendable, 10),
		storage:     make(map[string]string),
		tokenExpiry: time.Now(),
	}, nil
}

func (m *Messenger) start(ctx context.Context) {
	go m.startAccessTokenRefresher(ctx)
	go m.startMessageQueueMapScanner(ctx)
	go m.startMessageQueueHandler(ctx)
}

func (m *Messenger) startMessageQueueMapScanner(ctx context.Context) {
	logger.Debug("starting MessageQueueMapScanner")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(MQScanInterval):
			m.handleMessageQueue()
		}
	}
}

func (m *Messenger) startMessageQueueHandler(ctx context.Context) {
	logger.Debug("starting MessageQueueHandler")
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-m.mq:
			m.handleMessage(msg)
		}
	}
}

func (m *Messenger) startAccessTokenRefresher(ctx context.Context) {
	logger.Debug("starting AccessTokenRefresher")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if time.Now().Add(5 * time.Minute).After(m.tokenExpiry) {
				params := m.requireParams("clientId", "clientSecret")
				token, expireSec, err := getAccessToken(params["clientId"], params["clientSecret"])
				if err != nil {
					logger.Error("refresh access token failed", "err", err)
					continue
				}
				m.accessToken = token
				m.tokenExpiry = time.Now().Add(time.Duration(expireSec) * time.Second)
			}
			time.Sleep(time.Minute)
		}
	}
}

func (m *Messenger) enqueueMessage(msg Sendable) {
	conversationId := msg.OpenConversationId()
	q, ok := m.mqm.Get(conversationId)
	if !ok {
		q = queue.New[Sendable]()
		defer m.mqm.Put(conversationId, q)
	}
	q.Enqueue(msg)
}

func (m *Messenger) handleMessage(msg Sendable) {
	if time.Now().After(m.tokenExpiry) {
		logger.Error("failed to send message because access token was expired, re-add message to queue")
		m.enqueueMessage(msg)
		return
	}
	_, err := sendMessage(m.accessToken, msg)
	if err != nil {
		logger.Error("failed to send message, re-add message to queue", "err", err)
		m.enqueueMessage(msg)
		return
	}
	err = m.cachePut(msg)
	if err != nil {
		logger.Error("failed to cache message", "err", err)
	}
}

func (m *Messenger) handleMessageQueue() {
	m.mqm.Each(func(conversationId string, mq *queue.Queue[Sendable]) bool {
		prefix := fmt.Sprintf(CachePrefix, conversationId)
		for {
			count, err := m.cacheScan(prefix)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to scan prefix from badgerdb: %s", prefix), "err", err)
				return false
			}
			// 一分钟内发送出去的消息已经大于十条，则不发送
			if count >= 10 {
				return true
			}
			if !mq.Empty() {
				m.mq <- mq.Dequeue()
			}
		}
	})
}

func (m *Messenger) cacheScan(prefix string) (count int, err error) {
	err = m.cache.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		prefixBytes := []byte(prefix)
		iter.Seek(prefixBytes)

		count = 0
		for iter.ValidForPrefix(prefixBytes) {
			count++
			iter.Next()
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (m *Messenger) cachePut(msg Sendable) error {
	return m.cache.Update(func(txn *badger.Txn) error {
		cacheId := uuid.New().String()
		cacheKey := fmt.Sprintf(CachePrefix+"%s", msg.OpenConversationId(), cacheId)
		entry := badger.NewEntry([]byte(cacheKey), []byte(cacheId)).WithTTL(time.Minute)
		return txn.SetEntry(entry)
	})
}

func (m *Messenger) requireParams(keys ...string) (params map[string]string) {
	params = make(map[string]string)
	for _, key := range keys {
		val, ok := m.storage[key]
		if ok {
			params[key] = val
			continue
		}
		logger.Warn("can't get value from cached storage", "key", key)
	}
	return
}
