package ws

import (
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func WSHandler(hub *Hub) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
		}
		return c.Next()
	}
}

func WSUpgrader(hub *Hub) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		channel := c.Params("channel", "default")

		client := &Client{
			Connection: c,
			Send:       make(chan []byte, 256),
			Channel:    channel,
		}

		hub.Register(client)
		defer hub.Unregister(client)
		defer c.Close()

		readDone := make(chan struct{})

		go func() {
			defer close(readDone)
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					return
				}
				log.Printf("ws received: %s", string(message))
			}
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-readDone:
				return
			case message, ok := <-client.Send:
				if !ok {
					if err := c.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
						log.Printf("ws close error: %v", err)
					}
					return
				}
				if err := c.WriteMessage(websocket.TextMessage, message); err != nil {
					return
				}
			case <-ticker.C:
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}, websocket.Config{
		Subprotocols: []string{"chat"},
	})
}
