# Server api v2 compatibility

Source [docs](https://socket.io/docs/v2/server-api/)

## Server

| Status              | Source                             | Golang |
|---------------------|------------------------------------|--------|
| []                  | new Server(httpServer, options)    |        |
| []                  | new Server(port, options)          |        |
| []                  | new Server(options)                |        |
| []                  | server.sockets                     |        | 
| []                  | server.serveClient(value)          |        |
| []                  | server.path(value)                 |        |
| []                  | server.adapter(value)              |        |
| []                  | server.origins(value)              |        |
| []                  | server.origins(fn)                 |        |
| []                  | server.attach(httpServer, options) |        |
| []                  | server.attach(port, options)       |        |
| []                  | server.listen(httpServer, options) |        |
| []                  | server.listen(port, options)       |        |
| []                  | server.bind(engine)                |        |
| []                  | server.onconnection(socket)        |        |
| []                  | server.of(nsp)                     |        |
| []                  | server.close(callback)             |        |
| []                  | server.engine.generateId           |        |
| ---                 | ---                                | ---    |
| coverage: 0/17 - 0% |

## Namespace

| Status              | Source                             | Golang |
|---------------------|------------------------------------|--------|
| []                  | namespace.name                     |        |
| []                  | namespace.connected                |        |
| []                  | namespace.adapter                  |        |
| []                  | namespace.to(room)                 |        |
| []                  | namespace.in(room)                 |        |
| []                  | namespace.emit(eventName, ...args) |        |
| []                  | namespace.clients(callback)        |        |
| []                  | namespace.use(fn)                  |        |
| []                  | Event: 'connect'                   |        |
| []                  | Event: 'connection'                |        |
| []                  | Flag: 'volatile'                   |        |
| []                  | Flag: 'binary'                     |        |
| []                  | Flag: 'local'                      |        |
| ---                 | ---                                | ---    |
| coverage: 0/12 - 0% |

## Socket

| Status              | Source                                     | Golang |
|---------------------|--------------------------------------------|--------|
| []                  | socket.id                                  |        |
| []                  | socket.rooms                               |        |
| []                  | socket.client                              |        |
| []                  | socket.conn                                |        |
| []                  | socket.request                             |        |
| []                  | socket.handshake                           |        |
| []                  | socket.use(fn)                             |        |
| []                  | socket.send(...args)                       |        |
| []                  | socket.emit(eventName, ...args)            |        |
| []                  | socket.on(eventName, callback)             |        |
| []                  | socket.once(eventName, listener)           |        |
| []                  | socket.removeListener(eventName, listener) |        |
| []                  | socket.removeAllListeners(eventName)       |        |
| []                  | socket.eventNames()                        |        |
| []                  | socket.join(room, callback)                |        |
| []                  | socket.join(rooms, callback)               |        |
| []                  | socket.leave(room, callback)               |        |
| []                  | socket.to(room)                            |        |
| []                  | socket.in(room)                            |        |
| []                  | socket.compress(value)                     |        |
| []                  | socket.disconnect(close)                   |        |
| []                  | Flag: 'broadcast'                          |        |
| []                  | Flag: 'volatile'                           |        |
| []                  | Flag: 'binary'                             |        |
| []                  | Event: 'disconnect'                        |        |
| []                  | Event: 'error'                             |        |
| []                  | Event: 'disconnecting'                     |        |
| ---                 | ---                                        | ---    |
| coverage: 0/26 - 0% |                                            |

## Client

| Status             | Source         | Golang |
|--------------------|----------------|--------|
| []                 | client.conn    |        |
| []                 | client.request |        |
| ---                | ---            | ---    |
| coverage: 0/2 - 0% |                |