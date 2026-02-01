package ws

import (
	"common/redis"
	"context"
	"encoding/json"
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

	go readPong(client, r)
	go writePing(client) // TODO: починить на websocket отображение статуса в realtime
	log.Printf("User %s connected via WebSocket", userID)
}

func PubSubBlock() {
	pubsub := redis.EventsRedisClient().Subscribe(context.Background(), "block:events")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var event struct {
			Type      string `json:"type"`
			BlockerID string `json:"blocker_id"`
			BlockedID string `json:"blocked_id"`
			IsBlocked bool   `json:"is_blocked"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Invalid block event: %v", err)
			continue
		}

		// Рассылаем только заблокированному пользователю
		ClientsMu.RLock()
		blockedClient, exists := Clients[uuid.MustParse(event.BlockedID)]
		if exists && blockedClient.Conn != nil {
			blockedClient.mu.Lock()
			err := blockedClient.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			blockedClient.mu.Unlock()

			if err != nil {
				log.Printf("Failed to send block event to %s: %v", event.BlockedID, err)
			}
		}
		ClientsMu.RUnlock()
	}
}

func readPong(c *Client, r *repository.ProfileRepository) {
	defer func() {
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

func writePing(c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			if err := c.Conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(10*time.Second)); err != nil {
				log.Printf("Ping failed for user %s: %v", c.UserID, err)
				return
			}
			c.mu.Unlock()
		}
	}
}
