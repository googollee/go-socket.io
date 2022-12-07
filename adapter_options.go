package socketio

// RedisAdapterOptions is configuration to create new adapter
type RedisAdapterOptions struct {
	Addr     string
	Prefix   string
	Network  string
	Password string
	DB       int
}

func (ro *RedisAdapterOptions) getAddr() string {
	return ro.Addr
}

func defaultOptions() *RedisAdapterOptions {
	return &RedisAdapterOptions{
		Addr:    "127.0.0.1:6379",
		Prefix:  "socket.io",
		Network: "tcp",
	}
}

func getOptions(opts *RedisAdapterOptions) *RedisAdapterOptions {
	options := defaultOptions()

	if opts != nil {
		if opts.Addr != "" {
			options.Addr = opts.Addr
		}

		if opts.Prefix != "" {
			options.Prefix = opts.Prefix
		}

		if opts.Network != "" {
			options.Network = opts.Network
		}

		if opts.DB > 0 {
			options.DB = opts.DB
		}

		if len(opts.Password) > 0 {
			options.Password = opts.Password
		}
	}

	return options
}
