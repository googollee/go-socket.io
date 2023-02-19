## Debug mode

Golang socket.io support [uber zap](https://github.com/uber-go/zap) logger by global state from package `logger`.
This logger used in internal logic, and you could use into your logic.

## Default configuration from env:
```
GO_SOCKET_IO_LOG_LEVEL="error"
GO_SOCKET_IO_LOG_ENABLE="false"
GO_SOCKET_IO_DEBUG="false"
```

## Describe 

`GO_SOCKET_IO_LOG_LEVEL` - configure log level for server internal messages. <br/>
`GO_SOCKET_IO_LOG_ENABLE` - disable log output by one env. <br/>
`GO_SOCKET_IO_DEBUG` - this mode add stack trace for server debug. <br/>

## Examples:

### Set custom logger

Your custom logger must implement next interface:

```go
type Logger interface {
    Debugln(args ...interface{})
    Warnln(args ...interface{})
    Infoln(args ...interface{})
    Errorln(args ...interface{})
    Panicln(args ...interface{})
}
```

Override internal and get logger:

```go
import (
	...
	
    "github.com/googollee/go-socket.io/logger"
)


func main() {
    server := socketio.NewServer()
	
	log := logger.GetLogger()
    // update configuration
	//...
	logger.SetLogger(log)
}
```
