package engineio

type fakeServer struct {
	closed bool
}

func (f *fakeServer) Config() config {
	return config{
		PingTimeout:   60000 * time.Millisecond,
		PingInterval:  25000 * time.Millisecond,
		AllowRequest:  func(*http.Request) error { return nil },
		AllowUpgrades: true,
		Cookie:        "io",
	}
}

func (f *fakeServer) Transports() transportsType {
	t, _ := newTransportsType(nil)
	return t
}

func (f *fakeServer) OnClose() {
	f.closed = true
}
