package dingtalkbot

type Module interface {
	handlers() (middlewares []HandlerFunc, handler HandlerFunc)
}
