package socketio

type Broadcaster struct {
	Namespaces []*NameSpace
}

func (b *Broadcaster) Broadcast(name string, args ...interface{}) {
	for _, ns := range b.Namespaces {
		go ns.Emit(name, args...)
	}
}

func (b *Broadcaster) Except(namespace *NameSpace) *Broadcaster {
	for i, ns := range b.Namespaces {
		if ns == namespace {
			b.Namespaces = append(b.Namespaces[:i], b.Namespaces[i+1:]...)
			return b
		}
	}
	return b
}
