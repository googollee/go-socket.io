# go-socket.io

go-socket.io is library an implementation of [Socket.IO](http://socket.io) in Golang, which is a realtime application framework.

Current this library supports 1.4 version of the Socket.IO client. It supports room, namespaces and broadcast at now.

**Help wanted** This project is looking for contributors to help fix bugs and implement new features. Please check [Issue 192](https://github.com/googollee/go-socket.io/issues/192). All help is much appreciated.

## Badges

![Build Status](https://github.com/googollee/go-socket.io/workflows/CI/badge.svg)
[![GoDoc](http://godoc.org/github.com/googollee/go-socket.io?status.svg)](http://godoc.org/github.com/googollee/go-socket.io)
[![License](https://img.shields.io/github/license/golangci/golangci-lint)](/LICENSE)
[![Release](https://img.shields.io/github/release/googollee/go-socket.io.svg)](https://github.com/googollee/go-socket.io/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/googollee/go-socket.io)](https://goreportcard.com/report/github.com/googollee/go-socket.io)

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

Please check more examples into folder in project for details. [Examples](https://github.com/googollee/go-socket.io/tree/master/_examples)

## FAQ

It is some popular questions about this repository: 

- Is this library supported socket.io version 2?
    - No, but if you wanna you can help to do it. Join us in community chat Telegram   
- How to use go-socket.io with CORS?
    - Please see examples in [directory](https://github.com/googollee/go-socket.io/tree/master/_examples)

## Community

Telegram chat: [@go_socketio](https://t.me/go_socketio)

## Engineio

This project contains a sub-package called `engineio`. This used to be a separate package under https://github.com/googollee/go-engine.io.

It contains the `engine.io` analog implementation of the original node-package. https://github.com/socketio/engine.io It can be used without the socket.io-implementation. Please check the README.md in `engineio/`.

## License

The 3-clause BSD License  - see [LICENSE](https://opensource.org/licenses/BSD-3-Clause) for more details
