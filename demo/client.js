var url = 'ws://localhost:8080';
var eio = require('engine.io-client')(url, {
});

var msgCount = 0;

eio.on('open', function() {
    console.log('open '+url);
    eio.on('message', function(data) {
        if (data instanceof ArrayBuffer || data instanceof Buffer) {
            var a = new Uint8Array(data);
            console.log('receive: binary '+a.toString());
        } else {
            console.log('receive: text '+data);
        }
    });
    eio.on('upgrade', function() {
        console.log('upgrade');

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

    var text = 'hello';
    var ab = new ArrayBuffer(4);
    var a = new Uint8Array(ab);
    a.set([1,2,3,4]);

    console.log("sending: text "+text);
    eio.send(text);

    console.log("sending: binary 1,2,3,4");
    eio.send(ab);

    setInterval(function() {
        console.log("sending: text "+text);
        eio.send(text);

        console.log("sending: binary 1,2,3,4");
        eio.send(ab);
    }, 5*1000);
});
