package transport

type Type uint

const (
	Polling Type = iota
	Websocket
)

func (t Type) String() string {
	if t == Polling {
		return "polling"
	}
	if t == Websocket {
		return "websocket"
	}

	return "unspecified"
}
