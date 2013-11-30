package socketio

type Broadcaster struct {
  Namespaces []*NameSpace
}

func (b *Broadcaster) Broadcast(name string, args ...interface{}) {
  for _, ns := range b.Namespaces {
    go ns.Emit(name, args...)
  }
}
