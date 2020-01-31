package base

// FrameType is type of a message frame.
const (
	// TEXT is text type message.
	TEXT = FrameType(FrameString)
	// BINARY is binary type message.
	BINARY = FrameType(FrameBinary)
)
