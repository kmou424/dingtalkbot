package dingtalkbot

import (
	"fmt"
	"strings"
)

type Module interface {
	parseContext(message *Message) *Context
}

// Simple only one handler module
type Simple struct {
	handler HandlerFunc
}

func ModuleSimple() *Simple {
	return &Simple{}
}

func (s *Simple) Handle(handler HandlerFunc) *Simple {
	s.handler = handler
	return s
}

func (s *Simple) parseContext(message *Message) *Context {
	return &Context{
		Message:     message,
		middlewares: nil,
		handler:     s.handler,
		args:        []string{},
	}
}

// Chain multi handler module
type Chain struct {
	middlewares []HandlerFunc
	handler     HandlerFunc
}

func ModuleChain() *Chain {
	return &Chain{
		middlewares: []HandlerFunc{},
	}
}

func (c *Chain) Use(middleware HandlerFunc) *Chain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *Chain) Handle(handler HandlerFunc) *Chain {
	c.handler = handler
	return c
}

func (c *Chain) parseContext(message *Message) *Context {
	return &Context{
		Message:     message,
		middlewares: c.middlewares,
		handler:     c.handler,
		args:        []string{},
	}
}

// ChatChain check prefix as command handler module
type ChatChain struct {
	middlewares []HandlerFunc
	handlerMap  *RWMap[string, HandlerFunc]
	defHandler  HandlerFunc
}

func ModuleChatChain() *ChatChain {
	return &ChatChain{
		middlewares: []HandlerFunc{},
		handlerMap:  newRWMap[string, HandlerFunc](),
	}
}

func (c *ChatChain) formatPrefix(command string) string {
	return fmt.Sprintf("/%s", command)
}

func (c *ChatChain) Use(middleware HandlerFunc) *ChatChain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *ChatChain) Handle(command string, handler HandlerFunc) *ChatChain {
	c.handlerMap.Put(c.formatPrefix(command), handler)
	return c
}

func (c *ChatChain) Default(handler HandlerFunc) *ChatChain {
	c.defHandler = handler
	return c
}

func (c *ChatChain) parseContext(message *Message) *Context {
	if message.Type != TypeChat {
		return nil
	}
	content := strings.TrimLeft(message.Chat().Text.Content, " ")
	handler := c.defHandler
	args := []string{}
	c.handlerMap.Each(func(command string, h HandlerFunc) bool {
		// if command start with arguments, must is a space of suffix
		switch {
		case strings.HasPrefix(content, command+" "):
			args = strings.Split(content[len(command):], " ")
			handler = h
			return false
		case content == command:
			handler = h
			return false
		}
		return true
	})
	return &Context{
		Message:     message,
		middlewares: c.middlewares,
		handler:     handler,
		args:        args,
	}
}
