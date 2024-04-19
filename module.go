package dingtalkbot

type Module interface {
	handlers() (middlewares []HandlerFunc, handler HandlerFunc)
}

// Simple only one handler module
type Simple struct {
	handler HandlerFunc
}

func NewSimple() *Simple {
	return &Simple{}
}

func (s *Simple) Handle(handler HandlerFunc) *Simple {
	s.handler = handler
	return s
}

func (s *Simple) handlers() ([]HandlerFunc, HandlerFunc) {
	return nil, s.handler
}
