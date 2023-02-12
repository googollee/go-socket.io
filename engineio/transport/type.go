package transport

type Type int

const (
	Polling Type = iota
	Websocket
)

func GetType(req string) Type {
	if req == "polling" {
		return Polling
	}

	if req == "websocket" {
		return Websocket
	}

	return -1
}

func (t Type) String() string {
	if t == Polling {
		return "polling"
	}
	if t == Websocket {
		return "websocket"
	}

	return "unspecified"
}
