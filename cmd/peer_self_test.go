package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/types"
)

func setupMockDB() error {
	mockDB, err := initMockDB()
	if err != nil {
		return fmt.Errorf("failed to initialize mock db: %v", err)
	}

	err = resetMockDB(mockDB, types.Libp2pInfo{})
	if err != nil {
		return fmt.Errorf("failed to reset previous db tables: %v", err)
	}

	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	if err != nil {
		return fmt.Errorf("unable to auto migrate mock db: %v", err)
	}

	mockP2PInfo := types.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	if result.Error != nil {
		return fmt.Errorf("failed to insert data inside mock db: %v", err)
	}

	return nil
}

func Test_SelfPeerCmd(t *testing.T) {
	assert := assert.New(t)

	err := setupMockDB()
	assert.NoError(err)

	mockUtils := &MockUtilsService{}

	selfResponse := []byte(`{
    "ID": "abcdef12345",
    "Addrs": ["ip4/10000", "ip6/20000"]
    }`)
	mockUtils.SetResponseFor("GET", "/api/v1/peers/self", selfResponse)

	buf := new(bytes.Buffer)
	cmd := NewPeerSelfCmd(&backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.NoError(err)

	buf2 := new(bytes.Buffer)
	fmt.Fprintln(buf2, "Host ID: abcdef12345")
	fmt.Fprintln(buf2, "ip4/10000, ip6/20000")

	assert.Equal(buf.String(), buf2.String())
}
