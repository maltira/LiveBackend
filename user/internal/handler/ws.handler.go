package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"user/internal/repository"
	"user/pkg/redis"
	"user/pkg/utils"
	"user/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
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

	client := &websocket.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256), // буфер на 256 сообщений
	}

	websocket.ClientsMu.Lock()
	websocket.Clients[userID] = client
	websocket.ClientsMu.Unlock()

	utils.SetOnline(userID)

	go readPump(client, r)
	go writePump(client)
	log.Printf("User %s connected via WebSocket", userID)
}

// ? Подписки на события

func PubSubBlock() {
	pubsub := redis.UserRedis.Subscribe(context.Background(), "user:block:events")
	defer func() {
		_ = pubsub.Close()
	}()

	for msg := range pubsub.Channel() {
		var event utils.BlockEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Invalid block event: %v", err)
			continue
		}

		// Рассылаем только заблокированному пользователю
		websocket.ClientsMu.RLock()
		blockedClient, exists := websocket.Clients[uuid.MustParse(event.BlockedID)]
		if exists && blockedClient.Conn != nil {
			blockedClient.Mu.Lock()
			err := blockedClient.Conn.WriteMessage(ws.TextMessage, []byte(msg.Payload))
			blockedClient.Mu.Unlock()

			if err != nil {
				log.Printf("Failed to send block event to %s: %v", event.BlockedID, err)
			}
		}
		websocket.ClientsMu.RUnlock()
	}
}
func PubSubStatus() {
	pubsub := redis.UserRedis.Subscribe(context.Background(), "user:status:events")
	defer func() {
		_ = pubsub.Close()
	}()

	for msg := range pubsub.Channel() {
		var event utils.StatusEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			continue
		}

		websocket.ClientsMu.RLock()
		for _, client := range websocket.Clients {
			if client.Conn == nil {
				continue
			}
			client.Mu.Lock()
			_ = client.Conn.WriteMessage(ws.TextMessage, []byte(msg.Payload))
			client.Mu.Unlock()
		}
		websocket.ClientsMu.RUnlock()
	}
}
func PubSubNewMessage() {
	pubsub := redis.UserRedis.Subscribe(context.Background(), "chat:message:events")
	defer func() {
		_ = pubsub.Close()
	}()

	type MessageEvent struct {
		EventType    string   `json:"event_type"`
		ID           string   `json:"id"`
		ChatID       string   `json:"chat_id"`
		UserID       string   `json:"user_id"`
		Content      string   `json:"content"`
		Type         string   `json:"type"`
		CreatedAt    string   `json:"created_at"`
		Participants []string `json:"participants"`
	}
	for msg := range pubsub.Channel() {
		var event MessageEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			continue
		}

		for _, pID := range event.Participants {
			websocket.ClientsMu.RLock()
			client, exists := websocket.Clients[uuid.MustParse(pID)]
			if exists && client.Conn != nil {
				client.Mu.Lock()
				err := client.Conn.WriteMessage(ws.TextMessage, []byte(msg.Payload))
				client.Mu.Unlock()

				if err != nil {
					log.Printf("Failed to send new message to %s: %v", event.UserID, err)
				}
			}
			websocket.ClientsMu.RUnlock()
		}
	}
}

func readPump(c *websocket.Client, r *repository.ProfileRepository) {
	defer func() {
		// Удаляем клиента из карты
		websocket.ClientsMu.Lock()
		delete(websocket.Clients, c.UserID)
		websocket.ClientsMu.Unlock()

		utils.SetOffline(c.UserID, r)

		_ = c.Conn.Close()
		log.Printf("User %s disconnected", c.UserID)
	}()

	// устанавливаем дедлайн и обработчик pong (чтобы обнаружить разрыв)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		// клиент ответил pong - продлеваем deadline и онлайн
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		utils.SetOnline(c.UserID)
		return nil
	})
	c.Conn.SetReadLimit(512)

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil { // клиент отвалился
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("Unexpected WS close for user %s: %v", c.UserID, err)
			}
			break
		}
	}
}
func writePump(c *websocket.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Mu.Lock()
			if err := c.Conn.WriteControl(ws.PingMessage, []byte("ping"), time.Now().Add(10*time.Second)); err != nil {
				c.Mu.Unlock()
				log.Printf("Ping failed for user %s: %v", c.UserID, err)
				return
			}
			c.Mu.Unlock()
		}
	}
}
