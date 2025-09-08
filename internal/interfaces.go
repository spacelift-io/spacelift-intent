package internal

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

type ConnectionHandler interface {
	CloseConnections(ctx context.Context) error
	Handle(c *websocket.Conn, cookies []*http.Cookie) *WebsocketError
	GetAvailableExecutor() (string, error)
}
