package socketio

type NameSpace struct {
	*EventEmitter
	Name         string
	session      *Session
	onDisconnect func(*NameSpace)
}

func NewNameSpace(session *Session, name string) *NameSpace {
	return &NameSpace{session: session, Name: name, EventEmitter: NewEventEmitter()}
}

func (ns *NameSpace) OnDisconnect(fn func(*NameSpace)) {
	ns.onDisconnect = fn
}

func (ns *NameSpace) disconnected() {
	ns.onDisconnect(ns)
	ns.session.disconnected(ns)
}
