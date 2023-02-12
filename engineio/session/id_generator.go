package session

import (
	"strconv"
	"sync/atomic"
)

// Generator generates new session id. Default behavior is simple increasing number.
// If you need custom session id, for example using local ip as prefix, you can
// implement SessionIDGenerator and save in Configure. Engine.io will use custom
// one to generate new session id.
type Generator interface {
	NewID() string
}

type DefaultSessionGenerator struct {
	ID uint64
}

func NewSessionGenerator() *DefaultSessionGenerator {
	return &DefaultSessionGenerator{}
}

func (g *DefaultSessionGenerator) NewID() string {
	id := atomic.AddUint64(&g.ID, 1)

	return strconv.FormatUint(id, 36)
}
