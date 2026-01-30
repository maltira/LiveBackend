package ws

import (
	"log"
	"net/http"
	"time"
	"user/repository"
	"user/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Connect(c *gin.Context, r *repository.ProfileRepository) {
	userID := c.MustGet("userID").(uuid.UUID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WS upgrade failed for user %s: %v", userID, err)
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
	}

	ClientsMu.Lock()
	Clients[userID] = client
	ClientsMu.Unlock()

	if err = utils.SetOnline(userID); err != nil {
		log.Printf("Failed to set online for %s: %v", userID, err)
		_ = conn.Close()
		return
	}

	go readPump(client, r)
	go writePump(client)
	log.Printf("User %s connected via WebSocket", userID)
}

func readPump(c *Client, r *repository.ProfileRepository) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()

		// Удаляем клиента из карты
		ClientsMu.Lock()
		delete(Clients, c.UserID)
		ClientsMu.Unlock()

		if err := utils.SetOffline(c.UserID, r); err != nil {
			log.Printf("Failed to set offline for %s: %v", c.UserID, err)
		}

		_ = c.Conn.Close()
		log.Printf("User %s disconnected", c.UserID)
	}()

	// устанавливаем дедлайн и обработчик pong (чтобы обнаружить разрыв)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		// клиент ответил pong - продлеваем deadline и онлайн
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_ = utils.SetOnline(c.UserID)
		return nil
	})
	c.Conn.SetReadLimit(512)

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil { // клиент отвалился
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected WS close for user %s: %v", c.UserID, err)
			}
			break
		}
	}
}

func writePump(c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.Conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(10*time.Second)); err != nil {
				log.Printf("Ping failed for user %s: %v", c.UserID, err)
				return
			} else {
				log.Printf("Ping sent to user %s", c.UserID)
			}
		}
	}
}
