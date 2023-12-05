# go-websocket
go websocket

涵盖了websocket的基本使用，以及wss（ssl）的使用
功能包括：
1. 基本的websocket使用
2. wss（ssl）的使用
3. websocket的心跳检测
4. 广播消息，单播消息

1.启动方式

```go
go mod download

go run main.go
```

2.前端

ws.html 文件-->直接浏览器打开即可

3.wss（ssl）测试

若后端是https的方式启动，且只有测试证书的情况下

先下载nodejs: https://nodejs.org/en

```sh
npm install ws

node wss.js
```

