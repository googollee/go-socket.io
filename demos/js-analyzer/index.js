let sio3, sio2, main, map;

const handler = function (req, res) {
  console.log('handling url ' + req.url);

  if (req.url == '/socket.io.js.map') {
    res.setHeader('Content-Type', 'application/json');
    res.writeHead(200);
    res.end(map);
    return;
  }

  if (req.url == '/sio3.js') {
    res.setHeader('Content-Type', 'application/json');
    res.writeHead(200);
    res.end(sio3);
    return;
  }

  if (req.url == '/sio2.js') {
    res.setHeader('Content-Type', 'application/json');
    res.writeHead(200);
    res.end(sio2);
    return;
  }

  console.log('handling client');
  res.setHeader('Content-Type', 'text/html');
  res.writeHead(200);
  res.end(main);
};

const http = require('http').createServer(handler).
  listen(8000, 'localhost', () => {
  console.log('launched at 8000...');
});
const sio = require('socket.io')(http, {
  allowEIO3: true,
});

sio.on('connection', so => {
  console.log('connected ' + so.id)
  so.on('message', data => {
    console.log('message: ' + data);
    so.send(data);
    so.send(Buffer.from(data, 'utf-8'));
  })
  so.on('call', (arg, callback) => {
    console.log('call: ' + arg);
    callback('called '+arg);
  })
  so.on('disconnect', reason => {
    console.log('closed ' + so.id + ' reason: ' + reason);
  });
});

const fs = require('fs').promises;

fs.readFile(__dirname + '/node_modules/sio3/dist/socket.io.js.map')
  .then(contents => {
    map = contents;
}).catch(err => {
  console.error(`Could not read socket.io-client@3 socket.io.js.map file: ${err}`);
  process.exit(1);
});

fs.readFile(__dirname + '/node_modules/sio3/dist/socket.io.js')
  .then(contents => {
    sio3 = contents;
}).catch(err => {
  console.error(`Could not read socket.io-client@3 socket.io.js file: ${err}`);
  process.exit(1);
});

fs.readFile(__dirname + '/node_modules/sio2/dist/socket.io.js')
  .then(contents => {
    sio2 = contents;
}).catch(err => {
  console.error(`Could not read socket.io-client@2 socket.io.js file: ${err}`);
  process.exit(1);
});

fs.readFile(__dirname + '/main.html')
  .then(contents => {
    main = contents;
}).catch(err => {
  console.error(`Could not read main.html file: ${err}`);
  process.exit(1);
});


