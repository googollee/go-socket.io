package socketio

type Adapter interface {
	NewAdapter() (Broadcaster, error)
}

type Broadcaster interface {
	NewBroadcast(nsp string) Broadcast
}
