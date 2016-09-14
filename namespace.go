package socketio

// Namespace is the name space of a socket.io handler.
type Namespace interface {

	// Name returns the name of the namespace.
	Name() string

	// Of returns the namespace with given name.
	Of(name string) Namespace

	// On registers the function f to handle an event.
	On(event string, f interface{}) error

	// SetMux sets a new multiplexer for the handler.
	SetMux(mux EventHandler)
}

type namespace struct {
	*baseHandler
	root map[string]Namespace
}

func newNamespace(broadcast BroadcastAdaptor) *namespace {
	ret := &namespace{
		baseHandler: newBaseHandler("", broadcast),
		root:        make(map[string]Namespace),
	}
	ret.root[ret.Name()] = ret
	return ret
}

func (n *namespace) Name() string {
	return n.baseHandler.name
}

func (n *namespace) Of(name string) Namespace {
	if name == "/" {
		name = ""
	}
	if ret, ok := n.root[name]; ok {
		return ret
	}
	ret := &namespace{
		baseHandler: newBaseHandler(name, n.baseHandler.broadcast),
		root:        n.root,
	}
	n.root[name] = ret
	return ret
}
