
# gows

Minimal RFC compliant WebSocket Version 13 implementation in Pure Go
with zero dependencies and with `net/http` compatibility.

### Features

- RFC 6455 compliant WebSocket 13 implementation
- RFC 7602 compliant WebSocket Compression Extensions implementation

## Usage

### gows.Client

```go
import "github.com/cookiengineer/gows"
import "log"
import "time"

logger := log.New(os.Stdout, "[client] ", log.LstdFlags)
client, err0 := gows.NewClient("ws://localhost:8080")

if err0 == nil {

    err1 := client.Connect()

    if err1 == nil {

        time.Sleep(100 * time.Millisecond)
        client.Socket.Send([]byte("Hello, world!"))

        time.Sleep(1 * time.Second)
        client.Socket.Close(gows.StatusGoingAway, "Goodbye!")

    } else {
        logger.Fatal(err1)
    }

} else {
    logger.Fatal(err0)
}
```

### gows.Server

```go
import "github.com/cookiengineer/gows"
import "log"

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

- [simple](./examples/simple) is a simple client and server.
- [permessage-deflate](./examples/permessage-deflate) is a client and server that uses a websocket extension.
- [chatserver](./examples/chatserver) is a fully integrated web chat server.


## License

This library is licensed under the [X11/MIT License](./LICENSE.txt)

