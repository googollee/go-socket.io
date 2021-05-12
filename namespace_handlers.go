package socketio

import "sync"

type namespaceHandlers struct {
	handlers map[string]*namespaceHandler
	mu       sync.RWMutex
}

func newNamespaceHandlers() *namespaceHandlers {
	return &namespaceHandlers{
		handlers: make(map[string]*namespaceHandler),
	}
}

func (h *namespaceHandlers) Set(namespace string, handler *namespaceHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.handlers[namespace] = handler
}

func (h *namespaceHandlers) Get(nsp string) (*namespaceHandler, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	handler, ok := h.handlers[nsp]
	return handler, ok
}
