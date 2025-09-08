package websocket

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/pkg/errors"
)

type Client[T any, U any] struct {
	connection *Connection[T, U]
}

func NewClient[T any, U any]() *Client[T, U] {
	return &Client[T, U]{}
}

func (c *Client[T, U]) Connect(ctx context.Context, url string, id string, client *http.Client, httpHeaders http.Header) error {
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: client,
		HTTPHeader: httpHeaders,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Spacelift WebSocket: %w", err)
	}

	c.connection = NewConnection[T, U](id, conn)

	go c.connection.WriteLoop(ctx)
	go c.connection.ReadLoop(ctx)

	return nil
}

func (c *Client[T, U]) IsConnected() bool {
	return c.connection != nil && !c.connection.Closed()
}

func (c *Client[T, U]) Close(ctx context.Context) error {
	if c.connection == nil {
		return nil
	}
	return c.connection.Close(ctx)
}

func (c *Client[T, U]) CloseError() *websocket.CloseError {
	if c.connection == nil {
		return nil
	}
	closeError := c.connection.CloseError()
	return &closeError
}

func (c *Client[T, U]) SendMessage(ctx context.Context, message U) error {
	if c.connection == nil {
		return errors.New("client not connected")
	}
	return c.connection.Send(ctx, message)
}

func (c *Client[T, U]) ReceiveMessage(ctx context.Context) (*T, error) {
	if c.connection == nil {
		return nil, errors.New("client not connected")
	}
	var message T
	err := c.connection.Receive(ctx, &message)
	return &message, err
}
