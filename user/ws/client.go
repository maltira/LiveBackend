package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	mu     sync.Mutex
}

var (
	Clients   = make(map[uuid.UUID]*Client)
	ClientsMu sync.RWMutex
)

func IsClientOnline(clientID uuid.UUID) bool {
	ClientsMu.RLock()
	c, exists := Clients[clientID]
	ClientsMu.RUnlock()
	return exists && c.Conn != nil
}
