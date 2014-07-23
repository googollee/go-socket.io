package socketio

type Emitter interface {
	Emit(name string, args ...interface{})
	On(name string, f interface{})
}
