# go-engine.io

[![GoDoc](http://godoc.org/github.com/googollee/go-socket.io/engineio?status.svg)](http://godoc.org/github.com/googollee/go-socket.io/engineio)

go-engine.io is the implement of engine.io in golang, which is transport-based cross-browser/cross-device bi-directional communication layer for [go-socket.io](https://github.com/googollee/go-socket.io).

It is compatible with node.js implement, and supported long-polling and websocket transport.

## Install

Install the package with:

```bash
go get github.com/googollee/go-socket.io/engineio@v1
```

Import it with:

```go
import "github.com/googollee/go-socket.io/engineio"
```

and use `engineio` as the package name inside the code.

## Example

Please check example folder for details.

```go
package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/googollee/go-socket.io/engineio"
)

func main() {
	server := engineio.NewServer(nil)

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Fatalln("accept error:", err)
			}
			
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

The 3-clause BSD License  - see [LICENSE](https://opensource.org/licenses/BSD-3-Clause) for more details
