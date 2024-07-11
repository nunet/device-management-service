package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/db"
	"gitlab.com/nunet/device-management-service/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func (h *MockHandler) GetMetadataHandler(c *gin.Context) {
	metadata := models.Metadata{
		Name:            "metadata",
		UpdateTimestamp: 1633036800,
		Resource: struct {
			MemoryMax int64 `json:"memory_max,omitempty"`
			TotalCore int64 `json:"total_core,omitempty"`
			CPUMax    int64 `json:"cpu_max,omitempty"`
		}{
			MemoryMax: 16000,
			TotalCore: 8,
			CPUMax:    8,
		},
		Available: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			CPU:    4,
			Memory: 8000,
		},
		Reserved: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			CPU:    4,
			Memory: 8000,
		},
		Network:           "mainnet",
		PublicKey:         "abc123xyz",
		NodeID:            "node-001",
		AllowCardano:      true,
		NTXPricePerMinute: 0.1,
	}
	c.JSON(200, metadata)
}

func (h *MockHandler) ProvisionedCapacityHandler(c *gin.Context) {
	prov := models.Provisioned{
		CPU:      3.5,
		Memory:   16384,
		NumCores: 4,
	}
	c.JSON(200, prov)
}

func (h *MockHandler) CreatePaymentAddressHandler(c *gin.Context) {
	wallet := c.DefaultQuery("blockchain", "cardano")
	if wallet != "cardano" && wallet != "ethereum" {
		c.JSON(400, gin.H{"error": "invalid query data"})
		return
	}
	var addr, phrase string
	var resp models.BlockchainAddressPrivKey
	if wallet == "cardano" {
		addr = "abc123xyz"
		phrase = "barbarbarbar"
		resp = models.BlockchainAddressPrivKey{
			Address:  addr,
			Mnemonic: phrase,
		}
	} else {
		addr = "foobar123baz"
		phrase = "bazbazbazbaz"
		resp = models.BlockchainAddressPrivKey{
			Address:    addr,
			PrivateKey: phrase,
		}
	}
	c.JSON(200, resp)
}

func (h *MockHandler) OnboardHandler(c *gin.Context) {
	capacity := models.CapacityForNunet{
		ServerMode: true,
	}
	err := c.BindJSON(&capacity)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid request data"})
		return
	}
	metadata := models.Metadata{
		Name:            "foobar",
		UpdateTimestamp: 1625097600,
		Reserved: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			CPU:    capacity.CPU,
			Memory: capacity.Memory,
		},
		Network:           capacity.Channel,
		PublicKey:         "bazbazbaz",
		NodeID:            "foo123bar",
		AllowCardano:      capacity.Cardano,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
	}
	c.JSON(200, metadata)
}

func (h *MockHandler) OnboardStatusHandler(c *gin.Context) {
	status := models.OnboardingStatus{
		Onboarded:    true,
		Error:        nil,
		MachineUUID:  "foo",
		MetadataPath: "/.nunet/metadataV2.json",
		DatabasePath: "/.nunet/nunet.db",
	}
	c.JSON(200, status)
}

func (h *MockHandler) OffboardHandler(c *gin.Context) {
	query := c.DefaultQuery("force", "false")
	force, err := strconv.ParseBool(query)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid query data"})
		return
	}
	var msg string
	if force {
		msg = "forced offboard successfull"
	} else {
		msg = "offboard successfull"
	}
	c.JSON(200, gin.H{"message": msg})
}

func (h *MockHandler) ResourceConfigHandler(c *gin.Context) {
	if c.Request.ContentLength == 0 {
		c.JSON(400, gin.H{"error": "request body is empty"})
		return
	}

	var capacity models.CapacityForNunet
	err := c.BindJSON(&capacity)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid request data"})
		return
	}
	metadata := models.Metadata{
		Name:            "foobar",
		UpdateTimestamp: 1625097600,
		Reserved: struct {
			CPU    int64 `json:"cpu,omitempty"`
			Memory int64 `json:"memory,omitempty"`
		}{
			CPU:    capacity.CPU,
			Memory: capacity.Memory,
		},
		Network:           capacity.Channel,
		PublicKey:         "bazbazbaz",
		NodeID:            "foo123bar",
		AllowCardano:      capacity.Cardano,
		NTXPricePerMinute: capacity.NTXPricePerMinute,
	}
	c.JSON(200, metadata)
}

func TestGetMetadata(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.GET("/metadata", GetMetadataHandler)

	req, _ := http.NewRequest("GET", "/api/v1/onboarding/metadata", nil)
	w := httptest.NewRecorder()
	rServer.router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code, w.Body)

	var metadata *models.Metadata
	err := json.Unmarshal(w.Body.Bytes(), &metadata)
	assert.NoError(t, err)
}

func TestProvisionedCapacity(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.GET("/provisioned", ProvisionedCapacityHandler)

	req, _ := http.NewRequest("GET", "/api/v1/onboarding/provisioned", nil)
	w := httptest.NewRecorder()
	rServer.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, w.Body)

	var prov *models.Provisioned
	err := json.Unmarshal(w.Body.Bytes(), &prov)
	assert.NoError(t, err)
}

func TestCreatePaymentAddress(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.GET("/address/new", CreatePaymentAddressHandler)

	tests := []struct {
		description  string
		route        string
		query        string
		expectedCode int
	}{
		{
			description:  "cardano",
			query:        "?blockchain=cardano",
			expectedCode: 200,
		},
		{
			description:  "ethereum",
			query:        "?blockchain=ethereum",
			expectedCode: 200,
		},
		{
			description:  "empty blockchain query",
			query:        "",
			expectedCode: 200,
		},
	}
	for _, tc := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/onboarding/address/new"+tc.query, nil)
		rServer.router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, w.Body)

		var keypair *models.BlockchainAddressPrivKey
		err := json.Unmarshal(w.Body.Bytes(), &keypair)
		assert.NoError(t, err)
		if tc.description == "cardano" && tc.query == "" {
			assert.True(t, keypair.Mnemonic != "")
		} else if tc.description == "ethereum" {
			assert.True(t, keypair.PrivateKey != "")
		}
	}
}

func TestNotOnboardedOffboard(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.DELETE("/offboard", OffboardHandler)

	mockDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Errorf("unable to initialize mockdb: %v", err)
	}
	db.DB = mockDB

	type tests struct {
		description  string
		query        string
		expectedCode int
	}

	notOnboarded := []tests{
		{
			description:  "force query true",
			query:        "?force=true",
			expectedCode: 500,
		},
		{
			description:  "force query false",
			query:        "?force=false",
			expectedCode: 500,
		},
		{
			description:  "invalid force query",
			query:        "?force=foobar",
			expectedCode: 400,
		},
		{
			description:  "missing force query",
			query:        "",
			expectedCode: 500,
		},
	}

	for _, tc := range notOnboarded {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/onboarding/offboard"+tc.query, nil)
		rServer.router.ServeHTTP(w, req)

		assert.Equal(t, tc.expectedCode, w.Code, w.Body)
	}
}

// TODO test onboarded offboard,resourceConfig etc... when metadata file is deprecated

func TestOnboardStatusHandler(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.GET("/status", OnboardStatusHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/onboarding/status", nil)
	rServer.router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, w.Body)
}

func TestResourceConfigHandler(t *testing.T) {
	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.POST("/resource-config", ResourceConfigHandler)

	capacity := models.CapacityForNunet{ServerMode: true}
	bodyBytes, _ := json.Marshal(capacity)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/onboarding/resource-config", bytes.NewBuffer(bodyBytes))
	rServer.router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code, w.Body) // expect 500 because machine not onboarded
}
