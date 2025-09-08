package websocket

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pkg/errors"
)

var errConnectionAlreadyLocked = errors.New("connection already locked")

const (
	pongWait     = 60 * time.Second
	pingInterval = (pongWait * 9 / 10)
	writeTimeout = 10 * time.Second

	unknownClosureReason = "unknown closure reason"
)

type Connection[T any, U any] struct {
	id string
	ws *websocket.Conn

	readChan  chan T
	writeChan chan U
	closeChan chan struct{}

	locked atomic.Bool
	closed atomic.Bool

	closeReason websocket.CloseError
}

func NewConnection[T any, U any](id string, ws *websocket.Conn) *Connection[T, U] {
	ws.SetReadLimit(1024 * 1024) // 1MB
	return &Connection[T, U]{
		id:        id,
		ws:        ws,
		readChan:  make(chan T, 100),
		writeChan: make(chan U, 100),
		closeChan: make(chan struct{}),
	}
}

func (c *Connection[T, U]) ID() string {
	return c.id
}

func (c *Connection[T, U]) Lock() error {
	if !c.locked.CompareAndSwap(false, true) {
		return errConnectionAlreadyLocked
	}
	return nil
}

func (c *Connection[T, U]) Unlock() {
	c.locked.Store(false)
}

func (c *Connection[T, U]) IsLocked() bool {
	return c.locked.Load()
}

func (c *Connection[T, U]) Send(ctx context.Context, message U) error {
	if c.closed.Load() {
		return errors.New("connection is closed")
	}
	
	select {
	case c.writeChan <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closeChan:
		return errors.New("connection closed while sending")
	}
}

func (c *Connection[T, U]) Receive(ctx context.Context, message *T) error {
	select {
	case msg, ok := <-c.readChan:
		if !ok {
			return io.EOF
		}
		*message = msg
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (c *Connection[T, U]) Closed() bool {
	return c.closed.Load()
}

func (c *Connection[T, U]) CloseError() websocket.CloseError {
	return c.closeReason
}

func (c *Connection[T, U]) Close(ctx context.Context) error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(c.closeChan)
	return c.ws.Close(websocket.StatusNormalClosure, "normal closure")
}

func (c *Connection[T, U]) peerClosed(closeError websocket.CloseError) {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.closeReason = closeError
	close(c.closeChan)
}

func (c *Connection[T, U]) read(ctx context.Context) (T, error) {
	var message T
	err := wsjson.Read(ctx, c.ws, &message)
	if err != nil {
		return message, err
	}
	return message, nil
}

func (c *Connection[T, U]) write(ctx context.Context, message U) error {
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	err := wsjson.Write(writeCtx, c.ws, message)
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection[T, U]) ReadLoop(ctx context.Context) error {
	defer close(c.readChan)

	for {
		select {
		case <-c.closeChan:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := c.read(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				var closeError websocket.CloseError
				if errors.As(err, &closeError) {
					c.peerClosed(closeError)
					return nil
				}

				if errors.Is(err, io.EOF) {
					c.peerClosed(websocket.CloseError{
						Code:   websocket.StatusAbnormalClosure,
						Reason: "peer terminated connection without saying goodbye",
					})
					return nil
				}

				return errors.Wrap(err, "failed to read from websocket")
			}
			c.readChan <- message
		}
	}
}

func (c *Connection[T, U]) WriteLoop(ctx context.Context) error {
	// TODO: Make ping interval configurable
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()
	defer close(c.writeChan)

	for {
		select {
		case <-c.closeChan:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-pingTicker.C:
			if err := c.ping(ctx); err != nil {
				return errors.Wrap(err, "failed to ping websocket")
			}
		case message := <-c.writeChan:
			if err := c.write(ctx, message); err != nil {
				return errors.Wrap(err, "failed to write to websocket")
			}
		}
	}
}

func (c *Connection[T, U]) ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, pongWait)
	defer cancel()
	if err := c.ws.Ping(pingCtx); err != nil {
		return errors.Wrap(err, "failed to ping")
	}
	return nil
}
