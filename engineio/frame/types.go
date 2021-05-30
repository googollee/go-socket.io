package frame

import "fmt"

type Type int

const (
	Binary Type = iota
	Text
)

func (t Type) String() string {
	switch t {
	case Binary:
		return "binary"
	case Text:
		return "text"
	}

	return fmt.Sprintf("unknown(%d)", t)
}
