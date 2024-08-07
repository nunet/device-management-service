package cmd

import (
	"bytes"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/models"
)

type MockResources struct {
	provisioned *models.Provisioned
}

func (mr *MockResources) GetTotalProvisioned() *models.Provisioned {
	return mr.provisioned
}

func Test_CapacityCmdHasFlags(t *testing.T) {
	assert := assert.New(t)

	mockConn := &MockConnection{}
	mockResources := &MockResources{}
	mockUtils := &MockUtilsService{}

	cmd := NewCapacityCmd(mockConn, mockResources, mockUtils)

	assert.True(cmd.HasAvailableFlags())

	expectedFlags := []string{"full", "available", "onboarded"}

	flags := cmd.Flags()
	flags.VisitAll(func(f *flag.Flag) {
		assert.Contains(expectedFlags, f.Name)
	})
}

func Test_CapacityCmdWithoutFlags(t *testing.T) {
	assert := assert.New(t)

	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}
	mockResources := &MockResources{}
	expectedResponse := "Retrieve capacity of the machine, onboarded or available amount of resources\n"
	expectedResponse += "\nUsage:\n  capacity [flags]\n"
	expectedResponse += "\nFlags:\n"
	expectedResponse += "  -a, --available   display amount of resources still available for onboarding\n"
	expectedResponse += "  -f, --full        display device \n"
	expectedResponse += "  -h, --help        help for capacity\n"
	expectedResponse += "  -o, --onboarded   display amount of onboarded resources\n"

	cmd := NewCapacityCmd(mockConn, mockResources, mockUtils)

	buf := new(bytes.Buffer)

	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	assert.Nil(err)
	assert.Equal(expectedResponse, buf.String())
}

func Test_CapacityCmdFull(t *testing.T) {
	assert := assert.New(t)

	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	resources := &models.Provisioned{
		CPU:      10000,
		Memory:   10000,
		NumCores: 10,
	}
	mockResources := &MockResources{provisioned: resources}
	mockUtils := &MockUtilsService{}

	cmd := NewCapacityCmd(mockConn, mockResources, mockUtils)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--full"})

	err := cmd.Execute()
	assert.NoError(err)

	buf2 := new(bytes.Buffer)

	// write table to another buffer
	table := setupTable(buf2)
	handleFull(table, mockResources)
	table.Render()

	// compare buffers
	assert.Equal(buf.String(), buf2.String())
}

func Test_CapacityCmdAvailable(t *testing.T) {
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

	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockResources := &MockResources{}
	mockUtils := &MockUtilsService{}

	cmd := NewCapacityCmd(mockConn, mockResources, mockUtils)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--available"})

	err = cmd.Execute()
	assert.NoError(err)

	buf2 := new(bytes.Buffer)
	table := setupTable(buf2)

	err = handleAvailable(table, mockUtils)
	assert.NoError(err)

	table.Render()

	assert.Equal(buf.String(), buf2.String())
}

func Test_CapacityCmdOnboarded(t *testing.T) {
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

	conns := GetMockConn(true)
	mockConn := &MockConnection{conns: conns}
	mockUtils := &MockUtilsService{}
	mockResources := &MockResources{}

	cmd := NewCapacityCmd(mockConn, mockResources, mockUtils)

	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)

	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--onboarded"})

	err = cmd.Execute()
	assert.NoError(err)

	table := setupTable(buf2)

	err = handleOnboarded(table, mockUtils)
	assert.NoError(err)

	table.Render()

	assert.Equal(buf.String(), buf2.String())
}
