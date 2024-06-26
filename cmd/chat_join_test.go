package cmd

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"strconv"
// 	"testing"

// 	"github.com/stretchr/testify/assert"

// 	"gitlab.com/nunet/device-management-service/network/libp2p"
// )

// func Test_ChatJoinCmdInvalidArgs(t *testing.T) {
// 	assert := assert.New(t)

// 	// initialize placeholder chats so that program does not return error prematurely
// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWebSocket := &MockWebSocket{}

// 	cmd := NewChatJoinCmd(mockUtils, mockWebSocket)

// 	buf := new(bytes.Buffer)
// 	cmd.SetOut(buf)
// 	cmd.SetErr(buf)

// 	tests := []struct {
// 		name  string
// 		input []string
// 		want  string
// 	}{
// 		{name: "no args", input: []string{}, want: "no chat ID specified"},
// 		{name: "empty arg", input: []string{""}, want: "no chat ID specified"},
// 		{name: "multiple args", input: []string{"abc", "123", "efg"}, want: "unable to join multiple chats"},
// 		{name: "string args", input: []string{"abc"}, want: "argument is not integer"},
// 	}

// 	for _, tc := range tests {
// 		cmd.SetArgs(tc.input)
// 		err := cmd.Execute()
// 		assert.ErrorContains(err, tc.want)
// 	}
// }

// func Test_ChatJoinCmdWithoutChats(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWebSocket := &MockWebSocket{}

// 	cmd := NewChatJoinCmd(mockUtils, mockWebSocket)

// 	buf := new(bytes.Buffer)
// 	cmd.SetOut(buf)
// 	cmd.SetErr(buf)
// 	cmd.SetArgs([]string{"0"})

// 	err = cmd.Execute()
// 	assert.ErrorContains(err, "no incoming stream match chat ID specified")
// }

// func Test_ChatJoinCmdInitializeFail(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWebSocket := &MockWebSocket{initializeErr: fmt.Errorf("websocket not found")}

// 	for _, chat := range chatList {
// 		cmd := NewChatJoinCmd(mockUtils, mockWebSocket)
// 		cmd.SetArgs([]string{strconv.Itoa(chat.ID)})

// 		buf := new(bytes.Buffer)
// 		cmd.SetOut(buf)
// 		cmd.SetErr(buf)

// 		err = cmd.Execute()
// 		assert.ErrorContains(err, "websocket not found")
// 	}

// 	assert.Equal(mockWebSocket.initializeCalled, len(chatList))
// }

// func Test_ChatJoinCmdCloseOnEOF(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWebSocket := &MockWebSocket{}

// 	for _, chat := range chatList {
// 		cmd := NewChatJoinCmd(mockUtils, mockWebSocket)
// 		cmd.SetArgs([]string{strconv.Itoa(chat.ID)})

// 		buf := new(bytes.Buffer)
// 		cmd.SetOut(buf)
// 		cmd.SetErr(buf)

// 		err := cmd.Execute()
// 		assert.NoError(err)

// 		assert.Contains(buf.String(), "Error: EOF")
// 	}
// }

// func Test_ChatJoinCmdSuccess(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWS := &MockWebSocket{}

// 	for _, chat := range chatList {
// 		cmd := NewChatJoinCmd(mockUtils, mockWS)
// 		cmd.SetArgs([]string{strconv.Itoa(chat.ID)})

// 		buf := new(bytes.Buffer)
// 		cmd.SetOut(buf)
// 		cmd.SetErr(buf)

// 		err := cmd.Execute()
// 		assert.NoError(err)
// 	}

// 	assert.Equal(mockWS.initializeCalled, mockWS.closeCalled, mockWS.readMessageCalled, mockWS.writeMessageCalled, mockWS.pingCalled, len(chatList))
// }

// func Test_ChatJoinCmdReadMessage(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWS := &MockWebSocket{}

// 	tests := []struct {
// 		name string
// 		msg  []string
// 		want string
// 	}{
// 		{name: "single message with newline", msg: []string{"Message\n"}, want: "Peer: Message\n"},
// 		{name: "multiple messages with newline", msg: []string{"Message 1\n", "Message 2\n", "Message 3\n"}, want: "Peer: Message 1\nPeer: Message 2\nPeer: Message 3\n"},
// 		{name: "single newline", msg: []string{"\n"}, want: ""},
// 		{name: "multiple newlines", msg: []string{"\n\n\n\n"}, want: ""},
// 		{name: "empty string", msg: []string{""}, want: ""},
// 		{name: "single message", msg: []string{"Message"}, want: "Peer: Message\n"},
// 		{name: "multiple messages", msg: []string{"Message 1", "Message 2", "Message 3"}, want: "Peer: Message 1\nPeer: Message 2\nPeer: Message 3\n"},
// 		{name: "no messages", msg: []string(nil), want: "EOF"},
// 	}

// 	for _, tc := range tests {
// 		mockWS.readMessages = tc.msg

// 		cmd := NewChatJoinCmd(mockUtils, mockWS)
// 		cmd.SetArgs([]string{"0"})

// 		buf := new(bytes.Buffer)
// 		cmd.SetOut(buf)
// 		cmd.SetErr(buf)

// 		err := cmd.Execute()
// 		assert.NoError(err)

// 		if mockWS.readMessages != nil {
// 			assert.Equal(tc.want, buf.String())
// 		} else {
// 			assert.Contains(buf.String(), tc.want)
// 		}

// 		mockWS.readMessages = nil
// 		buf.Reset()
// 	}
// }

// func Test_ChatJoinCmdWriteMessage(t *testing.T) {
// 	assert := assert.New(t)

// 	chatList := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)

// 	mockUtils := &MockUtilsService{}
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	mockWS := &MockWebSocket{}

// 	tests := []struct {
// 		name string
// 		msg  string
// 		want []string
// 	}{
// 		{name: "single message", msg: "Message\n", want: []string{"Message"}},
// 		{name: "multiple messages", msg: "Message 1\nMessage 2\nMessage 3\n", want: []string{"Message 1", "Message 2", "Message 3"}},
// 		{name: "single newline", msg: "\n", want: []string(nil)},
// 		{name: "multiple newlines", msg: "\n\n\n\n", want: []string(nil)},
// 		{name: "empty string", msg: "", want: []string(nil)},
// 		{name: "no newline", msg: "Message", want: []string(nil)},
// 	}

// 	for _, tc := range tests {
// 		cmd := NewChatJoinCmd(mockUtils, mockWS)
// 		cmd.SetArgs([]string{"0"})

// 		in := bytes.NewBufferString(tc.msg)
// 		cmd.SetIn(in)

// 		out := new(bytes.Buffer)
// 		cmd.SetOut(out)
// 		cmd.SetErr(out)

// 		err := cmd.Execute()
// 		assert.NoError(err)

// 		assert.Equal(tc.want, mockWS.writtenMessages)

// 		mockWS.writtenMessages = nil
// 		out.Reset()
// 	}
// }
