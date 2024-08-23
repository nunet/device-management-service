package backend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocket struct {
	Conn *websocket.Conn
}

func (ws *WebSocket) Initialize(url string) error {
	var err error
	ws.Conn, _, err = websocket.DefaultDialer.Dial(url, nil)
	return err
}

func (ws *WebSocket) Close() error {
	return ws.Conn.Close()
}

func (ws *WebSocket) ReadMessage(ctx context.Context, w io.Writer) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, msg, err := ws.Conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseAbnormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure) ||
					strings.Contains(err.Error(), "use of closed network connection") {
					return fmt.Errorf("connection closed")
				}
				return fmt.Errorf("error reading message: %w", err)
			}
			fmt.Fprintf(w, "%s\n", msg)
		}
	}
}

func (ws *WebSocket) WriteMessage(ctx context.Context, r io.Reader) error {
	reader := bufio.NewReader(r)
	inputChan := make(chan string)

	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			inputChan <- msg
		}
		close(inputChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-inputChan:
			err := ws.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseAbnormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure) ||
					strings.Contains(err.Error(), "use of closed network connection") {
					return fmt.Errorf("connection closed")
				}
				return fmt.Errorf("error writing message: %w", err)
			}
		}
	}
}

func (ws *WebSocket) Ping(ctx context.Context, _ io.Writer) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := ws.Conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return fmt.Errorf("unable to send ping: %w", err)
			}
		}
	}
}
