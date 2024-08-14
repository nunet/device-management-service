package backend

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/peer"
	gonet "github.com/shirou/gopsutil/net"

	"gitlab.com/nunet/device-management-service/types"
)

type ResourceManager interface {
	GetTotalProvisioned() *types.Provisioned
}

// PeerManager abstracts libp2p functionality
type PeerManager interface {
	ClearIncomingChatRequests() error
	Decode(s string) (peer.ID, error)
}

type WalletManager interface {
	GetCardanoAddressAndMnemonic() (*types.BlockchainAddressPrivKey, error)
	GetEthereumAddressAndPrivateKey() (*types.BlockchainAddressPrivKey, error)
}

// NetworkManager abstracts connection on ports
type NetworkManager interface {
	GetConnections(kind string) ([]gonet.ConnectionStat, error)
}

// Utility abstracts helper functions under utils package
type Utility interface {
	IsOnboarded() (bool, error)
	ResponseBody(c *gin.Context, method, endpoint, query string, body []byte) ([]byte, error)
}

// WebSocketClient provides functionality to chat commands
type WebSocketClient interface {
	Initialize(url string) error
	Close() error
	ReadMessage(ctx context.Context, w io.Writer) error
	WriteMessage(ctx context.Context, r io.Reader) error
	Ping(ctx context.Context, w io.Writer) error
}

// Logger abstracts systemd journal entries
type Logger interface {
	AddMatch(match string) error
	Close() error
	GetEntry() (*sdjournal.JournalEntry, error)
	Next() (uint64, error)
}

// FileSystem abstracts Afero/os calls
type FileSystem interface {
	Create(name string) (FileHandler, error)
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (FileHandler, error)
	ReadFile(filename string) ([]byte, error)
	RemoveAll(path string) error
	Walk(root string, walkFn filepath.WalkFunc) error
}

// FileHandler abstracts file interfaces shared between "os" and "afero" so that both can be used interchangeably
type FileHandler interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Writer
	io.WriterAt
	Stat() (os.FileInfo, error)
	WriteString(s string) (int, error)
}
