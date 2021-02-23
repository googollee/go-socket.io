# go-socket.io

[![GoDoc](http://godoc.org/github.com/googollee/go-socket.io?status.svg)](http://godoc.org/github.com/googollee/go-socket.io) 
[![Build Status](https://travis-ci.org/googollee/go-socket.io.svg)](https://travis-ci.org/googollee/go-socket.io)
[![Coverage Status](https://coveralls.io/repos/github/googollee/go-socket.io/badge.svg?branch=v1.4)](https://coveralls.io/github/googollee/go-socket.io?branch=v1.4)
[![Go Report Card](https://goreportcard.com/badge/github.com/googollee/go-socket.io)](https://goreportcard.com/report/github.com/googollee/go-socket.io)

go-socket.io is library an implementation of [Socket.IO](http://socket.io) in Golang, which is a realtime application framework.

Current this library supports 1.4 version of the Socket.IO client. It supports room, namespaces and broadcast at now.

**Help wanted** This project is looking for contributors to help fix bugs and implement new features. Please check [Issue 192](https://github.com/googollee/go-socket.io/issues/192). All help is much appreciated.

## Contents

- [Install](#install)
- [Example](#example)
- [FAQ](#faq)
- [Engine.io](#engineio)
- [Community](#community)
- [License](#license)

## Install

Install the package with:

```bash
go get github.com/googollee/go-socket.io
```

Import it with:

```go
import "github.com/googollee/go-socket.io"
```

and use `socketio` as the package name inside the code.

## Example

Please check more examples into folder in project for details. [Examples](https://github.com/googollee/go-socket.io/tree/master/_example)


## How to use Redis broadcast adapter
```
server := socketio.NewServer(nil)
ok, err := server.Adapter(&socketio.RedisAdapterOptions{
    Host:   "127.0.0.1",
    Port:   "6379",
    Prefix: "socket.io",
})

if !ok {
    fmt.Println("error:", err)
    return
}
```

## FAQ

It is some popular questions about this repository: 

- Is this library supported socket.io version 2?
    - No, but if you wanna you can help to do it. Join us in community chat Telegram   
- How to use go-socket.io with CORS?
    - Please see examples in [directory](https://github.com/googollee/go-socket.io/tree/master/_example)
- What is minimal version Golang support for this library?
    - We required Go 1.9 or upper!
- How to user?
    - Go-socket.io compatibility with Socket.IO 0.9.x, please use branch 0.9.x * or tag go-socket.io@v0.9.1

## Community

Telegram chat: [@go_socketio](https://t.me/go_socketio)

## Engineio

This project contains a sub-package called `engineio`. This used to be a separate package under https://github.com/googollee/go-engine.io.

It contains the `engine.io` analog implementation of the original node-package. https://github.com/socketio/engine.io It can be used without the socket.io-implementation. Please check the README.md in `engineio/`.

## License

The 3-clause BSD License  - see LICENSE for more details
