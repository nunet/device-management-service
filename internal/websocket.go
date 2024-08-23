// Package internal is a work in progress. It is planned to accommodate
// modules such as db and types.
package internal

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// UpgradeConnection is generic protocol upgrader for entire DMS.
var UpgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(_ *http.Request) bool { return true },
}

// WebSocketConnection is pointer to gorilla/websocket.Conn
type WebSocketConnection struct {
	*websocket.Conn
}

// Command represents a command to be executed
type Command struct {
	Command string
	NodeID  string // ID of the node where command will be executed
	Result  string
	Conn    *WebSocketConnection
}

var commandChan = make(chan Command)

var clients = make(map[WebSocketConnection]string)

// ListenForWs listens to the connected client for any message. It is assumed that
// every message that is coming is a command to be executed.
func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			zlog.Sugar().Errorf("Error:", fmt.Sprintf("%v", r))
		}
	}()

	cmd := Command{NodeID: clients[*conn], Conn: conn}

	for {
		_, msg, err := conn.ReadMessage()
		if err == nil { // if NO error
			// logic to send command and fetch the output
			cmd.Command = string(msg)
			commandChan <- cmd
		}
	}
}

// SendCommandForExecution work is to send command for execution and fetch the result
// This function listens for new commands from commandChan
func SendCommandForExecution() {
	for {
		command := <-commandChan
		zlog.Sugar().Infof("%v", command)
		// TO BE IMPLEMENTED
		// send command

		// fetch result

		// send back result
		err := command.Conn.WriteMessage(websocket.TextMessage, []byte(command.Command))
		if err != nil {
			zlog.Sugar().Warnf("failed to write message: %w", err)
		}
	}
}
