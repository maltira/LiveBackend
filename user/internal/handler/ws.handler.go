package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"user/config"
	"user/internal/models/dto"
	"user/pkg/rabbitmq"
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

type Message struct {
	ID             uuid.UUID  `json:"id"`
	ChatID         uuid.UUID  `json:"chat_id"`
	UserID         *uuid.UUID `json:"user_id"` // null = система
	Content        string     `json:"content"`
	Type           string     `json:"type"`
	ReplyToMessage *uuid.UUID `json:"reply_to_message"`

	ReadBy []uuid.UUID `gorm:"type:uuid[]"`

	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at"`
}
type Participant struct {
	ID     uuid.UUID `json:"id"`
	ChatID uuid.UUID `json:"chat_id"`
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`

	JoinedAt   time.Time  `json:"joined_at"`
	MutedUntil *time.Time `json:"muted_until"` // для групп
}

func Connect(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

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

	go readPump(client)
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
		EventType      string    `json:"event_type"`
		ID             string    `json:"id"`
		ChatID         string    `json:"chat_id"`
		UserID         string    `json:"user_id"`
		Content        string    `json:"content"`
		Type           string    `json:"type"`
		CreatedAt      time.Time `json:"created_at"`
		ReplyToMessage string    `json:"reply_to_message"`
		Participants   []string  `json:"participants"`
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
				log.Println("Сообщение отправлено пользователю", pID)
				if err != nil {
					log.Printf("Failed to send new message to %s: %v", event.UserID, err)
				}
			}
			websocket.ClientsMu.RUnlock()
		}
	}
}
func PubSubReadAck() {
	pubsub := redis.UserRedis.Subscribe(context.Background(), "chat:read_ack:events")
	defer func() {
		_ = pubsub.Close()
	}()

	for msg := range pubsub.Channel() {
		var event struct {
			Event        string   `json:"event"`
			ChatID       string   `json:"chat_id"`
			UserID       string   `json:"user_id"`
			Participants []string `json:"participants"`
			MessageIDs   []string `json:"message_ids"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Invalid read_ack event: %v", err)
			continue
		}

		// Рассылаем пользователям
		broadcast, _ := json.Marshal(map[string]interface{}{
			"event_type":  "read_update",
			"chat_id":     event.ChatID,
			"user_id":     event.UserID,
			"message_ids": event.MessageIDs,
		})
		for _, p := range event.Participants {
			if p == event.UserID {
				continue
			}
			clientID, err := uuid.Parse(p)
			if err != nil {
				continue
			}
			websocket.ClientsMu.RLock()
			client, exists := websocket.Clients[clientID]
			if exists && client.Conn != nil {
				client.Mu.Lock()
				err = client.Conn.WriteMessage(ws.TextMessage, broadcast)
				client.Mu.Unlock()
				if err != nil {
					log.Printf("Failed to send read_update %s: %v", p, err)
				}
			}
			websocket.ClientsMu.RUnlock()
		}
	}
}

func readPump(c *websocket.Client) {
	defer func() {
		// Удаляем клиента из карты
		websocket.ClientsMu.Lock()
		delete(websocket.Clients, c.UserID)
		websocket.ClientsMu.Unlock()

		utils.SetOffline(c.UserID)

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
		_, msg, err := c.Conn.ReadMessage()
		if err != nil { // клиент отвалился
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				log.Printf("Unexpected WS close for user %s: %v", c.UserID, err)
			}
			break
		}

		// Парсим входящее сообщение
		var incoming struct {
			Type string `json:"type"` // message, typing, read, pong и т.д.
		}
		if err = json.Unmarshal(msg, &incoming); err != nil {
			log.Printf("Invalid JSON from %s: %v", c.UserID, err)
			continue
		}

		switch incoming.Type {
		case "read":
			var readEvent struct {
				ChatID     string   `json:"chat_id"`
				MessageIDs []string `json:"message_ids"`
			}
			if err = json.Unmarshal(msg, &readEvent); err != nil {
				continue
			}

			chatID, err := uuid.Parse(readEvent.ChatID)
			if err != nil {
				continue
			}

			// Публикуем событие, чтобы обновить read_by
			event := map[string]interface{}{
				"event":       "read_update",
				"user_id":     c.UserID.String(),
				"chat_id":     chatID.String(),
				"message_ids": readEvent.MessageIDs,
			}
			payload, _ := json.Marshal(event)
			err = rabbitmq.Publish("chat.events", payload)

			if err != nil {
				log.Printf("Failed to publish read_update event: %v", err)
			}
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
