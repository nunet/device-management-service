package cmd

import (
	"bytes"
	"fmt"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/models"
)

// checks if command has expected flags
func Test_PeerListCmdHasFlags(t *testing.T) {
	assert := assert.New(t)

	cmd := NewPeerListCmd(&backend.Utils{})

	assert.True(cmd.HasAvailableFlags())

	expectedFlags := []string{"dht"}

	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(expectedFlags, f.Name)
	})
}

// command output without passing flags
func Test_PeerListCmdNoFlag(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	responseBootstrap := []byte(`[
    {"ID": "fjaldkfjaslkfals", "Addrs": ["/ip4/v1/2000", "ip6/v2/3000"]},
    {"ID": "vncjnvxmcvvca", "Addrs": ["/ip4/v2/7888", "ip6/10000"]},
    {"ID": "aabbdbdbbdsaj", "Addrs": ["/ip4/20399", "ip6/88888"]}
    ]`)
	responseDHT := []byte(`[
    "jfalksdfjalsdkn",
    "q4uriq9e859349e",
    "09402fdvnvdbfya",
    "qreuiru7785890p"
    ]`)

	mockUtils := &MockUtilsService{}

	mockUtils.SetResponseFor("GET", "/api/v1/peers", responseBootstrap)
	mockUtils.SetResponseFor("GET", "/api/v1/peers/dht", responseDHT)

	buf := new(bytes.Buffer)
	cmd := NewPeerListCmd(&backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.NoError(err)

	buf2 := new(bytes.Buffer)
	peersBoot, err := getBootstrapPeers(buf2, &backend.Utils{})
	assert.NoError(err)
	peersDHT, err := getDHTPeers(&backend.Utils{})
	assert.NoError(err)

	fmt.Fprintf(buf2, "Bootstrap peers (%d)\n", len(peersBoot))
	for _, b := range peersBoot {
		fmt.Fprintf(buf2, "%s\n", b)
	}

	fmt.Fprintf(buf2, "\n")

	fmt.Fprintf(buf2, "DHT peers (%d)\n", len(peersDHT))
	for _, d := range peersDHT {
		fmt.Fprintf(buf2, "%s\n", d)
	}

	assert.Equal(buf.String(), buf2.String())
}

// command output when passing all flags
func Test_PeerListCmdWithFlags(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	responseDHT := []byte(`[
    "jfalksdfjalsdkn",
    "q4uriq9e859349e",
    "09402fdvnvdbfya",
    "qreuiru7785890p"
    ]`)

	mockUtils := &MockUtilsService{}

	mockUtils.SetResponseFor("GET", "/api/v1/peers/dht", responseDHT)

	buf := new(bytes.Buffer)
	cmd := NewPeerListCmd(&backend.Utils{})
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.NoError(err)

	peersDHT, err := getDHTPeers(&backend.Utils{})
	assert.NoError(err)

	buf2 := new(bytes.Buffer)
	fmt.Fprintf(buf2, "DHT peers (%d)\n", len(peersDHT))
	for _, d := range peersDHT {
		fmt.Fprintf(buf2, "%s\n", d)
	}

	assert.Equal(buf.String(), buf2.String())
}

// command output when received message 'no peers found'
func Test_PeerListCmdWithMessage(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	buf := new(bytes.Buffer)
	cmd := NewPeerListCmd(&backend.Utils{})
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.ErrorContains(err, "No peers found")
}

// command output if DHT array is empty
func Test_PeerListCmdEmptyDHTArray(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	// empty array
	responseDHT := []byte(`[]`)

	mockUtils := &MockUtilsService{}

	mockUtils.SetResponseFor("GET", "/api/v1/peers/dht", responseDHT)

	buf := new(bytes.Buffer)
	cmd := NewPeerListCmd(&backend.Utils{})
	cmd.SetArgs([]string{"--dht"})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.ErrorContains(err, "no DHT peers available")
}
