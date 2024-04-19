package dingtalkbot

type Module interface {
	handlers(message *Message) (middlewares []HandlerFunc, handler HandlerFunc)
}
