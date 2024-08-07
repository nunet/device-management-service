package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	gonet "github.com/shirou/gopsutil/net"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var mockFS = afero.NewMemMapFs()

// ========= MOCK IMPLEMENTATIONS ==========

type MockUtilsService struct {
	responses map[string][]byte
}

// SetResponseFor is a helper method. It sets a mock response for a specific method and endpoint
func (mu *MockUtilsService) SetResponseFor(method, endpoint string, resp []byte) {
	key := method + ":" + endpoint
	if mu.responses == nil {
		mu.responses = make(map[string][]byte)
	}

	mu.responses[key] = resp
}

func (mu *MockUtilsService) ResponseBody(c *gin.Context, method, endpoint, query string, body []byte) ([]byte, error) {
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

func (mc *MockConnection) GetConnections(kind string) ([]gonet.ConnectionStat, error) {
	return mc.conns, nil
}

// ========= HELPERS ==========

func initMockDB() (*gorm.DB, error) {
	// initialize mocked db in-memory
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return &gorm.DB{}, fmt.Errorf("error trying to initialize db: %v", err)
	}

	return mockDB, nil
}

func resetMockDB(mockDB *gorm.DB, schema models.Libp2pInfo) error {
	err := mockDB.Migrator().DropTable(schema)
	if err != nil {
		return fmt.Errorf("failed to drop tables: %v", err)
	}

	return nil
}

// ========= TESTS ==========

func Test_InfoCmd(t *testing.T) {
	assert := assert.New(t)

	mockDB, err := initMockDB()
	assert.NoError(err)

	// reset previous tables because of shared in-memory
	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
		PrivateKey: []byte("secretkey"),
	}

	// insert mocked data inside db
	result := mockDB.Create(&mockP2PInfo)
	assert.NoError(result.Error)

	// TODO: get info from onboarding db repo

	expectedResponse := "+----------------------+----------+\n"
	expectedResponse += "|         INFO         |  VALUE   |\n"
	expectedResponse += "+----------------------+----------+\n"
	expectedResponse += "| Name                 | metadata |\n"
	expectedResponse += "| Update Timestamp     |        0 |\n"
	expectedResponse += "| Memory Max           |      256 |\n"
	expectedResponse += "| Total Core           |        4 |\n"
	expectedResponse += "| CPU Max              |      700 |\n"
	expectedResponse += "| Available CPU        |      690 |\n"
	expectedResponse += "| Available Memory     |      246 |\n"
	expectedResponse += "| Reserved CPU         |       10 |\n"
	expectedResponse += "| Reserved Memory      |       10 |\n"
	expectedResponse += "| Network              | tcp      |\n"
	expectedResponse += "| Public Key           | abc123   |\n"
	expectedResponse += "| Node ID              |          |\n"
	expectedResponse += "| Allow Cardano        | false    |\n"
	expectedResponse += "| Dashboard            |          |\n"
	expectedResponse += "| NTX Price Per Minute | 0.000000 |\n"
	expectedResponse += "+----------------------+----------+\n"

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
	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	// initialize empty data
	emptyP2PInfo := models.Libp2pInfo{}

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
	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
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
	err = resetMockDB(mockDB, models.Libp2pInfo{})
	assert.NoError(err)

	// create table using Libp2pInfo struct
	err = mockDB.AutoMigrate(&models.Libp2pInfo{})
	assert.NoError(err)

	mockP2PInfo := models.Libp2pInfo{
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
