package main

import "github.com/cookiengineer/gows"
import "github.com/cookiengineer/gows/extensions"
import "log"
import "os"
import "time"

func main() {

	// WebSocket Server with permessage-deflate extension
	go func() {

		logger := log.New(os.Stdout, "[server] ", log.LstdFlags)

		server := &gows.Server{
			Addr: ":3000",
			Handler: func(websocket *gows.WebSocket) {

				logger.Print("Client connected!")

				if len(websocket.Extensions) > 0 {
					for _, ext := range websocket.Extensions {
						logger.Printf("Extension negotiated: %s", ext.Name())
					}
				}

				websocket.OnMessage = func(message []byte) {
					logger.Printf("Received message: %s", message)

					// Echo the message back
					websocket.Send(message)
				}

				websocket.OnClose = func() {
					logger.Print("Client disconnected!")
				}

			},
			Extensions: []gows.Extension{
				extensions.NewPermessageDeflate(),
			},
			ErrorLog: logger,
		}

		err := server.Listen()

		if err != gows.ErrServerClosed {
			logger.Fatal(err)
		}

	}()

	// WebSocket Client with permessage-deflate extension
	go func() {

		time.Sleep(100 * time.Millisecond)

		logger := log.New(os.Stdout, "[client] ", log.LstdFlags)

		client, err0 := gows.NewClient("ws://localhost:3000")

		if err0 == nil {

			client.Extensions = []gows.Extension{
				extensions.NewPermessageDeflate(),
			}

			err1 := client.Connect()

			if err1 == nil {

				if len(client.Socket.Extensions) > 0 {
					for _, ext := range client.Socket.Extensions {
						logger.Printf("Extension negotiated: %s", ext.Name())
					}
				} else {
					logger.Print("No extensions negotiated")
				}

				client.Socket.OnMessage = func(message []byte) {
					logger.Printf("Received message: %s", message)
				}

				// Send text message (compressed automatically)
				client.Socket.Send([]byte("Hello, world!"))

				time.Sleep(100 * time.Millisecond)

				// Send binary message (compressed automatically)
				client.Socket.SendBinary([]byte("Binary payload"))

				time.Sleep(100 * time.Millisecond)

				// Send a larger message to demonstrate compression benefit
				client.Socket.Send([]byte("This is a very very very very very very long message that would benefit a lot from compression because it is very very very very repetitive and each very amounts to at least 3 bytes saved!"))

				time.Sleep(1 * time.Second)

				client.Socket.Close(gows.StatusGoingAway, "Goodbye!")

			} else {
				logger.Fatal(err1)
			}

		} else {
			logger.Fatal(err0)
		}

	}()

	time.Sleep(3 * time.Second)

}
