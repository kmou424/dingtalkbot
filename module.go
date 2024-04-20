package dingtalkbot

type Module interface {
	handlers(message *Message) (middlewares []HandlerFunc, handler HandlerFunc)
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

func (s *Simple) handlers(_ *Message) ([]HandlerFunc, HandlerFunc) {
	return nil, s.handler
}

// Chain multi handler module
type Chain struct {
  middlewares []HandlerFunc
	handler HandlerFunc
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

func (c *Chain) handlers(_ *Message) ([]HandlerFunc, HandlerFunc) {
  return c.middlewares, c.handler
}