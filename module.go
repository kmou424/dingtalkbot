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
