// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// Package internal is a work in progress. It is planned to accommodate
// modules such as db and types.
package internal

import (
	"log"
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
			log.Printf("Recovered from panic: %v", r) // Log the panic if needed
		}
	}()

	cmd := Command{NodeID: clients[*conn], Conn: conn}

	for {
		_, msg, err := conn.ReadMessage()
		if err == nil {
			// logic to send command and fetch the output
			cmd.Command = string(msg)
			commandChan <- cmd
		} else {
			log.Printf("Error reading message: %v", err) // Handle the error if needed
			return
		}
	}
}

// SendCommandForExecution work is to send command for execution and fetch the result
// This function listens for new commands from commandChan
func SendCommandForExecution() {
	for {
		command := <-commandChan
		// TO BE IMPLEMENTED
		// send command

		// fetch result

		// send back result
		err := command.Conn.WriteMessage(websocket.TextMessage, []byte(command.Command))
		if err != nil {
			log.Printf("Error writing message: %v", err) // Log the error when message fails to send
		}
	}
}
