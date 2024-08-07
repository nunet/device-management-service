package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	repositories_gorm "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestSuite struct {
	afs      afero.Afero
	db       *gorm.DB
	WorkDir  string
	dbPath   string
	channels []string
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	afs := afero.Afero{
		Fs: afero.NewMemMapFs(),
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Errorf("could not open database: %v", err)
	}

	workDir := "/test"
	dbPath := "/test/db.db"

	channels := []string{"test1", "test2", "test3"}

	return &TestSuite{
		afs:      afs,
		db:       db,
		WorkDir:  workDir,
		dbPath:   dbPath,
		channels: channels,
	}
}

func (s *TestSuite) newTestOnboardingHandler() *OnboardingHandler {

	oConfig := onboarding.OnboardingConfig{
		Fs:             s.afs,
		P2PRepo:        repositories_gorm.NewLibp2pInfoRepository(s.db),
		UUIDRepo:       repositories_gorm.NewMachineUUIDRepository(s.db),
		AvResourceRepo: repositories_gorm.NewAvailableResourcesRepository(s.db),
		WorkDir:        s.WorkDir,
		DatabasePath:   s.dbPath,
		Channels:       s.channels,
	}
	service := onboarding.New(oConfig)
	return &OnboardingHandler{service: service}
}

func (s *TestSuite) setupDB() error {
	if s.db == nil {
		return fmt.Errorf("db not set")
	}
	s.db.AutoMigrate(&models.Libp2pInfo{})
	s.db.AutoMigrate(&models.AvailableResources{})
	s.db.AutoMigrate(&models.MachineUUID{})
	return nil
}

func (s *TestSuite) setupPrivateKey(key string) error {
	p2pInfo := &models.Libp2pInfo{
		PrivateKey: []byte(key),
	}
	return s.db.Create(&p2pInfo).Error
}

func (s *TestSuite) setupMachineUUID(uuid string) error {
	machine := models.MachineUUID{
		UUID: uuid,
	}
	return s.db.Create(&machine).Error
}

func TestOnboardStatus(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*TestSuite)
		expectedCode int
		expectedBody []byte
		wantErr      bool
	}{
		{
			name: "success",
			setupMock: func(ts *TestSuite) {
				ts.setupDB()
				ts.setupPrivateKey("abc123")
				ts.setupMachineUUID("12345")
			},
			expectedCode: 200,
		},
		{
			name: "fail",
			setupMock: func(ts *TestSuite) {
				ts.setupDB()
			},
			expectedCode: 500,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestSuite(t)
			handler := ts.newTestOnboardingHandler()

			if tt.setupMock != nil {
				tt.setupMock(ts)
			}

			router := gin.New()
			endpoint := "/api/v1/onboarding/status"
			router.GET(endpoint, handler.OnboardStatus)

			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedBody != nil {
				assert.Equal(t, tt.expectedBody, w.Body.Bytes())
			}

			if tt.wantErr {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestProvisionedCapacity(t *testing.T) {
	router := gin.New()
	ts := NewTestSuite(t)

	handler := ts.newTestOnboardingHandler()
	endpoint := "/api/v1/onboarding/provisioned"

	router.GET(endpoint, handler.ProvisionedCapacity)
	req := httptest.NewRequest("GET", endpoint, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, w.Body)

	var prov *models.Provisioned
	err := json.Unmarshal(w.Body.Bytes(), &prov)
	assert.NoError(t, err)
}

func TestCreatePaymentAddress(t *testing.T) {
	router := gin.New()
	ts := NewTestSuite(t)

	handler := ts.newTestOnboardingHandler()
	endpoint := "/api/v1/onboarding/address/new"

	router.GET(endpoint, handler.CreatePaymentAddress)

	tests := []struct {
		name         string
		query        string
		expectedCode int
		expectedBody []byte
		wantErr      bool
	}{
		{
			name:         "cardano",
			query:        "?blockchain=cardano",
			expectedCode: 200,
		},
		{
			name:         "ethereum",
			query:        "?blockchain=ethereum",
			expectedCode: 200,
		},
		{
			name:         "empty blockchain query",
			query:        "",
			expectedCode: 200,
		},
		{
			name:         "invalid wallet",
			query:        "?blockchain=test",
			expectedCode: 500,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", endpoint+tt.query, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if len(tt.expectedBody) > 0 {
				assert.Equal(t, tt.expectedBody, w.Body.Bytes())
			}
			if tt.wantErr {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestOnboard(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*TestSuite)
		reqBody      []byte
		expectedCode int
		expectedBody []byte
		wantErr      bool
	}{
		{
			name:         "invalid request data",
			reqBody:      []byte("invalid data"),
			expectedCode: 400,
			wantErr:      true,
		},
		{
			name:         "valid request data, internal error",
			reqBody:      []byte(`{"memory":1000,"cpu":1000,"channel":"test","payment_addr":"abc123"}`),
			expectedCode: 500,
			wantErr:      true,
		},
		// TODO: Add more error test cases
		// TODO: Check response body
		// TODO: Add success cases when ResourcesManager is done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestSuite(t)
			if tt.setupMock != nil {
				tt.setupMock(ts)
			}

			handler := ts.newTestOnboardingHandler()

			router := gin.New()
			endpoint := "/api/v1/onboarding/onboard"
			router.POST(endpoint, handler.Onboard)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", endpoint, bytes.NewBuffer(tt.reqBody))
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if len(tt.expectedBody) > 0 {
				assert.Equal(t, tt.expectedBody, w.Body.Bytes())
			}
			if tt.wantErr {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestOffboard(t *testing.T) {
	ts := NewTestSuite(t)
	handler := ts.newTestOnboardingHandler()

	router := gin.New()
	endpoint := "/api/v1/onboarding/offboard"
	router.DELETE(endpoint, handler.Offboard)

	tests := []struct {
		name         string
		setupMock    func(*TestSuite)
		query        string
		expectedCode int
		expectedBody []byte
		wantErr      bool
	}{
		{
			name: "internal error",
			setupMock: func(ts *TestSuite) {
				ts.setupDB()
				ts.setupPrivateKey("1234")
			},
			query:        "?force=false",
			expectedCode: 500,
			wantErr:      true,
		},
		{
			name:         "invalid request data",
			query:        "?force=12345",
			expectedCode: 400,
			wantErr:      true,
		},
		// TODO: Add more error test cases
		// TODO: Check response body
		// TODO: Add test case for code 200 when offboard is working
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(ts)
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("DELETE", endpoint+tt.query, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if len(tt.expectedBody) > 0 {
				assert.Equal(t, tt.expectedBody, w.Body.Bytes())
			}
			if tt.wantErr {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestResourceConfig(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*TestSuite)
		reqBody      []byte
		expectedCode int
		expectedBody []byte
		wantErr      bool
	}{
		{
			name:         "invalid request data",
			reqBody:      []byte("1234"),
			expectedCode: 400,
			wantErr:      true,
		},
		{
			name:         "empty request data",
			reqBody:      nil,
			expectedCode: 400,
			wantErr:      true,
		},
		{
			name:    "valid request data, not onboarded",
			reqBody: []byte(`{"memory":1000,"cpu":1000,"channel":"test","payment_addr":"abc123"}`),
			setupMock: func(ts *TestSuite) {
				ts.setupDB()
			},
			expectedCode: 500,
			wantErr:      true,
		},
		// TODO: Add more error cases
		// TODO: Check response body
		// TODO: Add cases for success when ResourcesManager is done
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestSuite(t)
			handler := ts.newTestOnboardingHandler()

			router := gin.New()
			endpoint := "/api/v1/onboarding/resource-config"
			router.POST(endpoint, handler.ResourceConfig)

			if tt.setupMock != nil {
				tt.setupMock(ts)
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", endpoint, bytes.NewBuffer(tt.reqBody))
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if len(tt.expectedBody) > 0 {
				assert.Equal(t, tt.expectedBody, w.Body.Bytes())
			}
			if tt.wantErr {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}
