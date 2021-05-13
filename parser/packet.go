package parser

// Type of packet.
type Type byte

const (
	// Connect type
	Connect Type = iota
	// Disconnect type
	Disconnect
	// Event type
	Event
	// Ack type
	Ack
	// Error type
	Error

	// BinaryEvent type
	binaryEvent
	// BinaryAck type
	binaryAck
)

// Header of packet.
type Header struct {
	Type      Type
	ID        uint64
	NeedAck   bool
	Namespace string
	Query     string
}

// Payload of packet.
type Payload struct {
	Header Header

	Data []interface{}
}
