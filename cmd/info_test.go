package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	gonet "github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MockUtilsService struct {
	responses map[string][]byte
	onboarded bool
	err       error
}

func (mu *MockUtilsService) IsOnboarded() (bool, error) {
	return mu.onboarded, mu.err
}

// SetResponseFor is a helper method. It sets a mock response for a specific method and endpoint
func (mu *MockUtilsService) SetResponseFor(method, endpoint string, resp []byte) {
	key := method + ":" + endpoint
	if mu.responses == nil {
		mu.responses = make(map[string][]byte)
	}

	mu.responses[key] = resp
}

func (mu *MockUtilsService) ResponseBody(_ *gin.Context, method, endpoint, _ string, _ []byte) ([]byte, error) {
	key := method + ":" + endpoint
	response, ok := mu.responses[key]

	if !ok {
		return nil, fmt.Errorf("no mock set for method: %s, endpoint: %s", method, endpoint)
	}

	return response, nil
}

type MockConnection struct {
	conns []gonet.ConnectionStat
}

func (mc *MockConnection) GetConnections(_ string) ([]gonet.ConnectionStat, error) {
	return mc.conns, nil
}

func initMockDB() (*gorm.DB, error) {
	// initialize mocked db in-memory
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return &gorm.DB{}, fmt.Errorf("error trying to initialize db: %v", err)
	}

	return mockDB, nil
}

// nolint
func resetMockDB(mockDB *gorm.DB, schema types.Libp2pInfo) error {
	err := mockDB.Migrator().DropTable(schema)
	if err != nil {
		return fmt.Errorf("failed to drop tables: %v", err)
	}

	return nil
}

func Test_InfoCmd(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	// reset previous tables because of shared in-memory
	err = resetMockDB(mockDB, types.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := types.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	// TODO: get info from onboarding db repo
	expectedResponse := `+----------------------+----------+\n
|         INFO         |  VALUE   |\n"
+----------------------+----------+\n"
| Name                 | metadata |\n"
| Update Timestamp     |        0 |\n"
| Memory Max           |      256 |\n"
| Total Core           |        4 |\n"
| CPU Max              |      700 |\n"
| Available CPU        |      690 |\n"
| Available Memory     |      246 |\n"
| Reserved CPU         |       10 |\n"
| Reserved Memory      |       10 |\n"
| Network              | tcp      |\n"
| Public Key           | abc123   |\n"
| Node ID              |          |\n"
| Allow Cardano        | false    |\n"
| Dashboard            |          |\n"
| NTX Price Per Minute | 0.000000 |\n"
+----------------------+----------+\n"`

	conns := GetMockConn(true)

	mockConn := &MockConnection{conns: conns}

	cmd := NewInfoCmd(mockConn, &backend.Utils{})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("error executing command: %v", err)
	}

	expected := new(bytes.Buffer)

	assert.Equal(expectedResponse, buf.String())

	buf.Reset()
	expected.Reset()
}

func Test_InfoCmdNotOnboarded(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	// reset previous tables because of shared in-memory
	err = resetMockDB(mockDB, types.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	assert.NoError(err)

	// initialize empty data
	emptyP2PInfo := types.Libp2pInfo{}

	// insert empty data inside db
	result := mockDB.Create(&emptyP2PInfo)
	assert.NoError(result.Error)

	conns := GetMockConn(true)

	mockConn := &MockConnection{conns: conns}

	buf := new(bytes.Buffer)
	cmd := NewInfoCmd(mockConn, &backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.ErrorContains(err, "not onboarded")
}

func Test_InfoCmdInvalidMetadata(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	// reset previous tables because of shared in-memory
	err = resetMockDB(mockDB, types.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := types.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	if result.Error != nil {
		t.Fatalf("could not add mocked data inside db: %v", err)
	}

	conns := GetMockConn(true)

	mockConn := &MockConnection{conns: conns}

	buf := new(bytes.Buffer)
	cmd := NewInfoCmd(mockConn, &backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.ErrorContains(err, "cannot read file")
}

func Test_InfoCmdDMSNotRunning(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	// reset previous tables because of shared in-memory
	err = resetMockDB(mockDB, types.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := types.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	if result.Error != nil {
		t.Fatalf("could not add mocked data inside db: %v", err)
	}

	conns := GetMockConn(false)

	mockConn := &MockConnection{conns: conns}

	buf := new(bytes.Buffer)
	cmd := NewInfoCmd(mockConn, &backend.Utils{})
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err = cmd.Execute()
	assert.ErrorContains(err, "looks like DMS is not running...")
}
