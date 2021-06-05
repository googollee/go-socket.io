let eio4, eio3, main;

const handler = function (req, res) {
  console.log('handling url ' + req.url);

  if (req.url == '/eio3.js') {
    res.setHeader('Content-Type', 'application/json');
    res.writeHead(200);
    res.end(eio3);
    return;
  }

  if (req.url == '/eio4.js') {
    res.setHeader('Content-Type', 'application/json');
    res.writeHead(200);
    res.end(eio4);
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
const eio = require('engine.io').attach(http, {
  allowEIO3: true,
});

eio.on('connection', so => {
  console.log('connected ' + so.id)
  so.on('message', (data) => {
    console.log('message: ' + data);
    so.send(data);
    so.send(Buffer.from(data, 'utf-8'));
  })
  so.on('close', () => {
    console.log('closed ' + so.id);
  });
});

const fs = require('fs').promises;

fs.readFile(__dirname + '/node_modules/eio4/dist/engine.io.js')
  .then(contents => {
    eio4 = contents;
}).catch(err => {
  console.error(`Could not read engine.io-client@4 engine.io.js file: ${err}`);
  process.exit(1);
});

fs.readFile(__dirname + '/node_modules/eio3/engine.io.js')
  .then(contents => {
    eio3 = contents;
}).catch(err => {
  console.error(`Could not read engine.io-client@3 engine.io.js file: ${err}`);
  process.exit(1);
});

fs.readFile(__dirname + '/main.html')
  .then(contents => {
    main = contents;
}).catch(err => {
  console.error(`Could not read main.html file: ${err}`);
  process.exit(1);
});


