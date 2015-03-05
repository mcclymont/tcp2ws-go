# tcp2ws-go
TCP to WebSocket Client proxy written in Go

This was written to solve the problem of communicating with a VNC server that is wrapped in WebSockets,
with a connecting client that doesn't speak WebSockets (any normal desktop client)

It creates a TCP server on the specified HOST/PORT combination, and each incoming connection gets
proxied to the upstream WebSocket destination

## Install
TODO: Install instructions

## Usage
```
go run tcp2ws.go --lhost= localhost:3333 --rhost ws://1.2.3.4:80/websock
```
#### Options:
  * -lhost="localhost:3333": TCP HOST:PORT to listen on
  * -rhost="REQUIRED": URL of upstream websocket server. Format is 'ws://HOST:PORT/PATH'
