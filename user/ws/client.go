package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
}

var (
	Clients   = make(map[uuid.UUID]*Client)
	ClientsMu sync.RWMutex
)
