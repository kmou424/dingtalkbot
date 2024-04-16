package dingtalkbot

type HandlerFunc[T any] func(*Context[T])

type ChatHandlerFunc HandlerFunc[ChatMessage]

type EventHandlerFunc HandlerFunc[EventMessage]
