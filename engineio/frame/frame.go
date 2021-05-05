package frame

// Type is the type of frames.
type Type byte

const (
	// String identifies a string frame.
	String Type = iota
	// Binary identifies a binary frame.
	Binary
)

// ByteToFrameType converts a byte to FrameType.
func ByteToFrameType(b byte) Type {
	return Type(b)
}

// Byte returns type in byte.
func (t Type) Byte() byte {
	return byte(t)
}
