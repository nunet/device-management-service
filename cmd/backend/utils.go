package backend

import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/gin-gonic/gin"

	"gitlab.com/nunet/device-management-service/utils"
)

type Utils struct{}

func (u *Utils) IsOnboarded() (bool, error) {
	body, err := u.ResponseBody(nil, "GET", "/api/v1/onboarding/status", "", nil)
	if err != nil {
		return false, fmt.Errorf("unable to get response body: %w", err)
	}
	onboarded, err := jsonparser.GetBoolean(body, "onboarded")
	if err != nil {
		return false, fmt.Errorf("failed to get 'onboarded' parameter from json response: %w", err)
	}

	return onboarded, nil
}

func (u *Utils) ResponseBody(c *gin.Context, method, endpoint, query string, body []byte) ([]byte, error) {
	return utils.ResponseBody(c, method, endpoint, query, body)
}
