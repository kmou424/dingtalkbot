package dingtalkbot

import (
	"fmt"

	"github.com/charmbracelet/log"
)

type HandlerFunc func(*Context)

type Context struct {
	*Message

	Client *Client

	index       int
	middlewares []HandlerFunc
	handler     HandlerFunc

	Next func()
}

func (c *Context) Abort() {
	c.AbortWithError(nil)
}

func (c *Context) AbortWithError(err error) {
	panic(err)
}

func (c *Context) dealWithMiddlewares() {
	c.Next = func() {
		c.index++
		if c.middlewares == nil {
			return
		}
		if c.index < len(c.middlewares) {
			c.middlewares[c.index](c)
		}
	}
	c.index = -1
	c.Next()
}

func (c *Context) handling() (err error) {
	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case error:
				err = e
			default:
				err = fmt.Errorf("%v", e)
			}
		}
	}()

	if c.handler == nil {
		return nil
	}

	logger.Info(fmt.Sprintf("Handling %s %s", c.Message.Type, func() string {
		switch c.Message.Type {
		case ChatType:
			return (*c.Message.Chat()).MsgId
		case EventType:
			return (*c.Message.Event()).Header.EventId
		}
		return "unknown"
	}()))

	c.dealWithMiddlewares()
	// if all middlewares are called
	if c.index == len(c.middlewares) {
		c.handler(c)
	}

	return nil
}

func (c *Context) Logger() *log.Logger {
	return logger
}
