
# gowebsocket

Minimal RFC 6455 compliant WebSocket Version 13 implementation in Pure Go
with zero dependencies and with `net/http` compatibility.


## Usage

### gowebsocket.Client

```go
logger := log.New(os.Stdout, "[client] ", log.LstdFlags)
client, err := gows.NewClient("ws://localhost:8080")

if err == nil {

    logger.Print(client)

    time.Sleep(100 * time.Millisecond)
    err := client.Socket.Send([]byte("Hello, world!"))
    fmt.Println(err)

    time.Sleep(100 * time.Millisecond)
    client.Socket.Send([]byte("Second Hello, world!"))

    time.Sleep(1 * time.Second)
    client.Socket.Close(gows.StatusGoingAway, "Goodbye!")

} else {
    logger.Fatal(err)
}
```

### gowebsocket.Server

```go
logger := log.New(os.Stdout, "[server] ", log.LstdFlags)
server := &gows.Server{
    Addr:    ":8080",
    Handler: func(websocket *gows.WebSocket) {

        logger.Print("Client connected!")

        websocket.OnMessage = func(message []byte) {
            logger.Printf("Received message: %s", message)
        }

        websocket.OnClose = func() {
            logger.Print("Client disconnected!")
        }

    },
    TLSConfig: nil,
    ErrorLog:  logger,
}

server.Listen()
```


## Examples

Check out the [examples](./examples) folder for further API usage examples.

- [chatserver](./examples/chatserver) is a simple chat server.
- [simple](./examples/simple) is a simple client and server.


## License

This library is licensed under the [X11/MIT License](./LICENSE.txt)

