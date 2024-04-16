package dingtalkbot

import (
	"fmt"
	"github.com/charmbracelet/log"
)

type Context[T any] struct {
	Message T
	*Bot

	index    int
	handlers []HandlerFunc[T]
}

func (c *Context[T]) Next() {
	c.index++
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}
}

func (c *Context[T]) Abort() {
	c.AbortWithError(nil)
}

func (c *Context[T]) AbortWithError(err error) {
	panic(err)
}

func (c *Context[T]) start() (err error) {
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
	c.index = -1
	c.Next()
	return nil
}

func (c *Context[T]) Logger() *log.Logger {
	return logger
}
