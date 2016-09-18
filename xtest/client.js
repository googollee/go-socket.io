var url = 'ws://localhost:8080';
var eio = require('engine.io-client')(url);

var msgCount = 0;

eio.on('open', function() {
    console.log('open '+url);
    eio.on('message', function(data) {
        console.log('receive: '+data);
        switch (msgCount) {
        case 0:
            if (data != "hello") {
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
    eio.on('upgrade', function() {
        console.log('upgrade');

        console.log("sending text");
        eio.send('hello');

        var ab = new ArrayBuffer(4);
        var a = new Uint8Array(ab);
        a.set([1,2,3,4]);
        console.log("sending binary");
        eio.send(ab);
    });
    eio.on('ping', function() {
        console.log('ping');
    });
    eio.on('pong', function() {
        console.log('pong');
    })
    eio.on('close', function() {
        console.log('close');
    });
    eio.on('error', function(err) {
        console.log('error: '+err);
        process.exit(-1);
    });
});
