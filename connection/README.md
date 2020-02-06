# go-engine.io

[![GoDoc](http://godoc.org/github.com/googollee/go-engine.io?status.svg)](http://godoc.org/github.com/googollee/go-engine.io) [![Build Status](https://travis-ci.org/googollee/go-engine.io.svg)](https://travis-ci.org/googollee/go-engine.io)
[![Coverage Status](https://coveralls.io/repos/github/googollee/go-engine.io/badge.svg?branch=v1.4)](https://coveralls.io/github/googollee/go-engine.io?branch=v1.4)

go-engine.io is the implement of engine.io in golang, which is transport-based cross-browser/cross-device bi-directional communication layer for [go-socket.io](https://github.com/googollee/go-socket.io).

It is compatible with node.js implement, and supported long-polling and websocket transport.

## Install

Install the package with:

```bash
go get github.com/googollee/go-engine.io@v1
```

Import it with:

```go
import "github.com/googollee/go-engine.io"
```

and use `engineio` as the package name inside the code.

## Example

Please check example folder for details.

```go
package main

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/googollee/go-engine.io"
)

func main() {
	server, _ := engineio.NewServer(nil)

	go func() {
		for {
			conn, _ := server.Accept()
			go func() {
				defer conn.Close()
				for {
					t, r, _ := conn.NextReader()
					b, _ := ioutil.ReadAll(r)
					r.Close()

					w, _ := conn.NextWriter(t)
					w.Write(b)
					w.Close()
				}
			}()
		}
	}()

	http.Handle("/engine.io/", server)
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
```

## License

The 3-clause BSD License  - see LICENSE for more details
