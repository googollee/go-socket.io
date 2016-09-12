var engine = require('engine.io');
var server = engine.listen(8080);

var msgCount = 0;

server.on('connection', function(socket) {
    console.log('open');
    socket.on('message', function(data) {
        if (data instanceof ArrayBuffer || data instanceof Buffer) {
            var a = new Uint8Array(data);
            console.log('receive: binary '+a.toString());
        } else {
            console.log('receive: text '+data);
        }
        socket.send(data);
    });
    socket.on('upgrade', function() {
        console.log('upgrade');
    });
    socket.on('ping', function() {
        console.log('ping');
    });
    socket.on('pong', function() {
        console.log('pong');
    })
    socket.on('close', function() {
        console.log('close');
    });
    socket.on('error', function(err) {
        console.log('error: '+err);
        process.exit(-1);
    });

});
