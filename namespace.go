package socketio

type Namespace interface {
	Name() string
	Of(namespace string) Namespace
	On(message string, f interface{}) error
	BroadcastTo(room, message string, args ...interface{}) error
}

type namespace struct {
	*baseHandler
	name string
	root map[string]Namespace
}

func newNamespace() *namespace {
	ret := &namespace{
		baseHandler: newBaseHandler(),
		name:        "",
		root:        make(map[string]Namespace),
	}
	ret.root[ret.name] = ret
	return ret
}

func (n *namespace) Name() string {
	return n.name
}

func (n *namespace) Of(name string) Namespace {
	if name == "/" {
		name = ""
	}
	if ret, ok := n.root[name]; ok {
		return ret
	}
	ret := &namespace{
		baseHandler: newBaseHandler(),
		name:        name,
		root:        n.root,
	}
	n.root[name] = ret
	return ret
}
