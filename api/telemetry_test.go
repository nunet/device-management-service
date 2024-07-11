package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/models"
)

func (h *MockHandler) GetFreeResourcesHandler(c *gin.Context) {
	free := models.FreeResources{
		BaseDBModel: models.BaseDBModel{
			ID: "1",
		},
		TotCpuHz:          3400000000,
		PriceCpu:          0.0005,
		Ram:               16384,
		PriceRam:          0.0001,
		Vcpu:              8,
		Disk:              500.0,
		PriceDisk:         0.0002,
		NTXPricePerMinute: 0.05,
	}
	c.JSON(200, free)
}

func TestGetFreeResourcesHandler(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/telemetry/free", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var free *models.FreeResources
	err := json.Unmarshal(w.Body.Bytes(), &free)
	assert.NoError(t, err)
}
