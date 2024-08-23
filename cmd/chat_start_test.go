package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockWebSocket struct {
	mu sync.Mutex

	initializeCalled int
	initializeErr    error

	closeCalled int
	closeErr    error

	readMessageCalled int
	readMessages      []string
	readMessageErr    error

	writeMessageCalled int
	writtenMessages    []string
	writeMessageErr    error

	pingCalled int
	pingErr    error
}

func (mws *MockWebSocket) Initialize(_ string) error {
	mws.initializeCalled++
	return mws.initializeErr
}

func (mws *MockWebSocket) Close() error {
	mws.closeCalled++
	return mws.closeErr
}

func (mws *MockWebSocket) ReadMessage(_ context.Context, w io.Writer) error {
	mws.readMessageCalled++
	if mws.readMessageErr != nil {
		return mws.readMessageErr
	}

	if len(mws.readMessages) == 0 {
		return io.EOF
	}

	for _, msgStr := range mws.readMessages {
		msg := strings.TrimSpace(msgStr)

		if msg != "" {
			fmt.Fprintf(w, "Peer: %s\n", msg)
		}
	}

	return nil
}

func (mws *MockWebSocket) WriteMessage(_ context.Context, r io.Reader) error {
	mws.writeMessageCalled++
	if mws.writeMessageErr != nil {
		return mws.writeMessageErr
	}

	reader := bufio.NewReader(r)
	for {
		msgStr, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		msg := strings.TrimSpace(msgStr)

		if msg != "" {
			mws.writtenMessages = append(mws.writtenMessages, strings.TrimSpace(msg))
		}
	}
}

func (mws *MockWebSocket) Ping(_ context.Context, _ io.Writer) error {
	mws.mu.Lock()
	defer mws.mu.Unlock()

	mws.pingCalled++
	return mws.pingErr
}

func Test_ChatStartCmdInvalidArgs(t *testing.T) {
	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}
	mockWebSocket := &MockWebSocket{}

	cmd := NewChatStartCmd(mockP2P, mockUtils, mockWebSocket)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{name: "no args", input: []string{}, want: "no peer ID specified"},
		{name: "empty args", input: []string{""}, want: "no peer ID specified"},
		{name: "multiple args", input: []string{"123", "abc", "efg"}, want: "cannot start multiple chats"},
		{name: "invalid peer ID", input: []string{"abcdefghi"}, want: "invalid peer ID"},
	}

	for _, tc := range tests {
		cmd.SetArgs(tc.input)

		err := cmd.Execute()
		assert.ErrorContains(t, err, tc.want)
	}
}

func Test_ChatStartCmdInitializeFail(t *testing.T) {
	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}
	mockWebSocket := &MockWebSocket{initializeErr: fmt.Errorf("websocket not found")}

	cmd := NewChatStartCmd(mockP2P, mockUtils, mockWebSocket)
	cmd.SetArgs([]string{"Qm12345abcdef"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()

	assert.Equal(t, mockWebSocket.initializeCalled, 1)
	assert.ErrorContains(t, err, "websocket not found")
}

func Test_ChatStartCmdCloseOnEOF(t *testing.T) {
	assert := assert.New(t)

	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}
	mockWebSocket := &MockWebSocket{}

	cmd := NewChatStartCmd(mockP2P, mockUtils, mockWebSocket)
	cmd.SetArgs([]string{"Qm12345abcdef"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.NoError(err)

	assert.Contains(buf.String(), "Error: EOF")
}

func Test_ChatStartCmdGoroutinesFail(t *testing.T) {
	assert := assert.New(t)

	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}

	mocks := []*MockWebSocket{
		{readMessageErr: fmt.Errorf("impossible to read")},
		{writeMessageErr: fmt.Errorf("impossible to write")},
		{pingErr: fmt.Errorf("impossible to interrupt")},
	}

	tests := []struct {
		name string
		mock *MockWebSocket
		want string
	}{
		{name: "read error", mock: mocks[0], want: "impossible to read"},
		{name: "write error", mock: mocks[1], want: "impossible to write"},
		{name: "interrupt", mock: mocks[2], want: "impossible to interrupt"},
	}

	for _, tc := range tests {
		cmd := NewChatStartCmd(mockP2P, mockUtils, tc.mock)
		cmd.SetArgs([]string{"Qm12345abcdef"})

		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.NoError(err)

		assert.Contains(buf.String(), tc.want)
	}
}

func Test_ChatStartCmdReadMessage(t *testing.T) {
	assert := assert.New(t)

	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}
	mockWS := &MockWebSocket{}

	tests := []struct {
		name string
		msg  []string
		want string
	}{
		{name: "single message with newline", msg: []string{"Message\n"}, want: "Peer: Message\n"},
		{name: "multiple messages with newline", msg: []string{"Message 1\n", "Message 2\n", "Message 3\n"}, want: "Peer: Message 1\nPeer: Message 2\nPeer: Message 3\n"},
		{name: "single newline", msg: []string{"\n"}, want: ""},
		{name: "multiple newlines", msg: []string{"\n\n\n\n"}, want: ""},
		{name: "empty string", msg: []string{""}, want: ""},
		{name: "single message", msg: []string{"Message"}, want: "Peer: Message\n"},
		{name: "multiple messages", msg: []string{"Message 1", "Message 2", "Message 3"}, want: "Peer: Message 1\nPeer: Message 2\nPeer: Message 3\n"},
		{name: "no messages", msg: []string(nil), want: "EOF"},
	}

	for _, tc := range tests {
		mockWS.readMessages = tc.msg

		cmd := NewChatStartCmd(mockP2P, mockUtils, mockWS)
		cmd.SetArgs([]string{"Qm12345abcdef"})

		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.NoError(err)

		if mockWS.readMessages != nil {
			assert.Equal(tc.want, buf.String())
		} else {
			assert.Contains(buf.String(), tc.want)
		}

		mockWS.readMessages = nil
		buf.Reset()
	}
}

func Test_ChatStartCmdWriteMessage(t *testing.T) {
	assert := assert.New(t)

	mockP2P := &MockP2PService{}
	mockUtils := &MockUtilsService{}
	mockWS := &MockWebSocket{}

	tests := []struct {
		name string
		msg  string
		want []string
	}{
		{name: "single message", msg: "Message\n", want: []string{"Message"}},
		{name: "multiple messages", msg: "Message 1\nMessage 2\nMessage 3\n", want: []string{"Message 1", "Message 2", "Message 3"}},
		{name: "single newline", msg: "\n", want: []string(nil)},
		{name: "multiple newlines", msg: "\n\n\n\n", want: []string(nil)},
		{name: "empty string", msg: "", want: []string(nil)},
		{name: "no newline", msg: "Message", want: []string(nil)},
	}

	for _, tc := range tests {
		cmd := NewChatStartCmd(mockP2P, mockUtils, mockWS)
		cmd.SetArgs([]string{"Qm12345abcdef"})

		in := bytes.NewBufferString(tc.msg)
		cmd.SetIn(in)

		out := new(bytes.Buffer)
		cmd.SetOut(out)
		cmd.SetErr(out)

		err := cmd.Execute()
		assert.NoError(err)

		assert.Equal(tc.want, mockWS.writtenMessages)

		mockWS.writtenMessages = nil
		out.Reset()
	}
}
