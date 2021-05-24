package packet

type Type int

const (
	PacketOpen Type = iota
	PacketClose
	PacketPing
	PacketPong
	PacketMessage
	PacketUpgrade
	PacketNoop
)
