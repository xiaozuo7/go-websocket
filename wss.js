const WebSocket = require('ws');

const ws = new WebSocket('wss://localhost:8080/ws?username=admin', {
    rejectUnauthorized: false,
});

ws.on('open', function open() {
    console.log('WebSocket connected');
    // 在这里可以发送一些测试数据
    ws.send('Hello from Node.js!');
});

ws.on('message', function incoming(data) {
    console.log('Received message: ' + data);
});

ws.on('error', function error(err) {
    console.error('WebSocket error: ' + err);
});

ws.on('close', function close() {
    console.log('WebSocket connection closed');
});
