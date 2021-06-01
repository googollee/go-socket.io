package packet

type Type int

const (
	Open Type = iota
	Close
	Ping
	Pong
	Message
	Upgrade
	Noop
)
