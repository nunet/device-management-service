package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func Test_ChatListCmd(t *testing.T) {
// 	assert := assert.New(t)

// 	mockUtils := &MockUtilsService{}

// 	chats := []libp2p.OpenStream{
// 		{ID: 0, StreamID: "abc", FromPeer: "addr1_abcde", TimeOpened: "12:00"},
// 		{ID: 1, StreamID: "def", FromPeer: "addr1_fghij", TimeOpened: "13:00"},
// 		{ID: 2, StreamID: "ghi", FromPeer: "addr1_klmno", TimeOpened: "18:00"},
// 	}

// 	var chatList []libp2p.OpenStream
// 	for _, chat := range chats {
// 		chatList = append(chatList, chat)
// 	}

// 	chatListJSON, err := json.Marshal(chatList)
// 	assert.NoError(err)

// 	chatResponse := []byte(chatListJSON)
// 	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", chatResponse)

// 	buf := new(bytes.Buffer)

// 	cmd := NewChatListCmd(mockUtils)
// 	cmd.SetOut(buf)
// 	cmd.SetErr(buf)

// 	err = cmd.Execute()
// 	assert.NoError(err)

// 	buf2 := new(bytes.Buffer)

// 	table := setupChatTable(buf2)
// 	for _, chat := range chatList {
// 		table.Append([]string{strconv.Itoa(chat.ID), chat.StreamID, chat.FromPeer, chat.TimeOpened})
// 	}
// 	table.Render()

// 	assert.Equal(buf.String(), buf2.String())
// }

func Test_ChatListCmdNoChats(t *testing.T) {
	mockUtils := &MockUtilsService{}

	errResponse := []byte(`{"error": "no incoming message stream"}`)
	mockUtils.SetResponseFor("GET", "/api/v1/peers/chat", errResponse)

	buf := new(bytes.Buffer)
	cmd := NewChatListCmd(mockUtils)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "no incoming message stream")
}
