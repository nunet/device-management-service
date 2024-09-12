package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	repositories_gorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/network/libp2p"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"gitlab.com/nunet/device-management-service/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestRouter creates a new Gin engine in test mode
// TODO: Make it agnostic to Gin to make tests more maintainable
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// performRequest is a helper function to create a request and record the response
func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestInitializeRoutes(t *testing.T) {
	// Create a new RESTServer instance
	logger := &logger.Logger{}
	p2p := &libp2p.Libp2p{}
	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.Libp2pInfo{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.FreeResources{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.OnboardedResources{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.ResourceAllocation{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.VirtualMachine{})
	assert.NoError(t, err)
	err = mockDB.AutoMigrate(&types.Services{})
	assert.NoError(t, err)

	resourceManager := resources.NewResourceManager(resources.ManagerRepos{
		FreeResources:      repositories_gorm.NewFreeResources(mockDB),
		OnboardedResources: repositories_gorm.NewOnboardedResources(mockDB),
		ResourceAllocation: repositories_gorm.NewResourceAllocation(mockDB),
	})

	oConf := onboarding.Config{
		Fs:              afero.Afero{Fs: afero.NewMemMapFs()},
		P2PRepo:         repositories_gorm.NewLibp2pInfo(mockDB),
		ResourceManager: resourceManager,
		UUIDRepo:        repositories_gorm.NewMachineUUID(mockDB),
		AvResourceRepo:  repositories_gorm.NewAvailableResources(mockDB),
		WorkDir:         "/test",
		DatabasePath:    "/test/db.sqlite",
		Channels:        []string{"test1", "test2", "test3"},
	}
	ob := onboarding.New(&oConf)
	mid := []gin.HandlerFunc{}
	port := uint32(8080)
	server := NewRESTServer(&RESTServerConfig{
		P2P:        p2p,
		Onboarding: ob,
		Resource:   resourceManager,
		Logger:     logger,
		MidW:       mid,
		Port:       port,
	})

	// Initialize the routes
	server.InitializeRoutes()

	origins := []struct {
		origin  string
		expCode int
	}{
		{origin: "http://localhost:8080", expCode: http.StatusForbidden},
		{origin: "http://localhost:8080", expCode: http.StatusForbidden},
		{origin: "http://localhost:8080", expCode: http.StatusForbidden},
		{origin: "http://localhost:9991", expCode: http.StatusOK},
		{origin: "http://localhost:9992", expCode: http.StatusOK},
	}

	for _, test := range origins {
		req, _ := http.NewRequest("GET", "/api/v1/onboarding/provisioned", nil)
		req.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, test.expCode, w.Code)
	}
}
