var engine = require('engine.io');
var server = engine.listen(8080);

var msgCount = 0;

server.on('connection', function(socket) {
    console.log('open');
    socket.on('message', function(data) {
        console.log('receive: '+data);
        switch (msgCount) {
        case 0:
            if (data != "hello你好") {
                process.exit(-1);
            }
            break;
        case 1:
            var a = new Uint8Array(data);
            if (a !== undefined) {
                process.exit(-1);
            }
            if (a.toString() != "1,2,3,4") {
                process.exit(-1);
            }
            break;
        default:
            process.exit(-1);
            break;
        }
    });
    socket.on('upgrade', function() {
        console.log('upgrade');

        console.log("sending text");
        socket.send('hello你好');

        var ab = new ArrayBuffer(4);
        var a = new Uint8Array(ab);
        a.set([1,2,3,4]);
        console.log("sending binary");
        socket.send(ab);
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
