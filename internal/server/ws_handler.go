package server

import (
	"context"
	"log"
	"net/http"
	"spacelift-intent-mcp/internal"

	"github.com/coder/websocket"
)

type WebsocketHandler interface {
	CloseConnections(ctx context.Context) error
	Handle(w http.ResponseWriter, r *http.Request)
}

type websocketHandler struct {
	ctx     context.Context
	handler internal.ConnectionHandler
}

func NewWebsocketHandler(ctx context.Context, handler internal.ConnectionHandler) WebsocketHandler {
	return &websocketHandler{ctx: ctx, handler: handler}
}

func (h *websocketHandler) CloseConnections(ctx context.Context) error {
	return h.handler.CloseConnections(ctx)
}

func (h *websocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// TODO: Add WebSocket origin validation for security
	// TODO: Add rate limiting to prevent connection spam
	// TODO: Add authentication/authorization before accepting WebSocket
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("failed to accept websocket: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to accept websocket"))
		return
	}

	defer c.CloseNow()
	wsErr := h.handler.Handle(c, r.Cookies())
	if wsErr != nil {
		w.WriteHeader(wsErr.Code)
		w.Write([]byte(wsErr.Message))
		return
	}
}
