package dingtalkbot

import (
	"fmt"

	"github.com/charmbracelet/log"
)

type Context[T any] struct {
	Message T
	Bot IBot

	index       int
	middlewares []HandlerFunc[T]
	handler     HandlerFunc[T]

	Next func()
}

func (c *Context[T]) Abort() {
	c.AbortWithError(nil)
}

func (c *Context[T]) AbortWithError(err error) {
	panic(err)
}

func (c *Context[T]) dealWithMiddlewares() {
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

func (c *Context[T]) handling() (err error) {
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

	c.dealWithMiddlewares()
	if c.index != len(c.middlewares) {
		return nil
	}

	c.handler(c)

	return nil
}

func (c *Context[T]) Logger() *log.Logger {
	return logger
}
