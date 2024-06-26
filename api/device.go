package api

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DeviceStatusHandler  godoc
//
//	@Summary		Retrieve device status
//	@Description	Retrieve device status whether paused/offline (unable to receive job deployments) or online
//	@Tags			device
//	@Produce		json
//	@Success		200	{object}	object
//	@Failure		500	{object}	object	"host node has not yet been initialized"
//	@Failure		500	{object}	object	"could not retrieve data from peer"
//	@Failure		500	{object}	object	"failed to type assert peer data for peer ID"
//	@Router			/device/status [get]
func DeviceStatusHandler(c *gin.Context) {
	// TODO: handle this after refactor
	// status, err := libp2p.DeviceStatus()
	// if err != nil {
	// 	c.AbortWithStatusJSON(500, gin.H{"error": "could not retrieve device status"})
	// 	return
	// }
	// c.JSON(200, gin.H{"online": status})
	c.AbortWithStatusJSON(500, gin.H{"error": "device status not implemented"})

}

// ChangeDeviceStatusHandler  godoc
//
//	@Summary		Change device status between online/offline
//	@Description	Change device status to online (able to receive jobs) or offline (unable to receive jobs).
//	@Tags			device
//	@Produce		json
//	@Failure		400	{object}	object	"empty content data"
//	@Failure		400	{object}	object	"invalid payload data"
//	@Failure		500	{object}	object	"host node has not yet been initialized"
//	@Failure		500	{object}	object	"could not retrieve data from self peer"
//	@Failure		500	{object}	object	"failed to type assert peer data for peer ID"
//	@Failure		500	{object}	object	"Failed to retrieve libp2p info from database"
//	@Failure		500	{object}	object	"Failed to update libp2p info in database"
//	@Failure		500	{object}	object	"failed to put peer data into peerstore"
//	@Success		200	{object}	object	"Device status successfully changed to online"
//	@Success		200	{object}	object	"Device status successfully changed to offline"
//	@Success		200	{object}	object	"no change in device status"
//	@Router			/device/status [post]
func ChangeDeviceStatusHandler(c *gin.Context) {
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("URL", "/device/status"))
	span.SetAttributes(attribute.String("MachineUUID", utils.GetMachineUUID()))

	var status struct {
		IsAvailable bool `json:"is_available"`
	}

	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(400, gin.H{"error": "empty content data"})
		return
	}
	err := c.ShouldBindJSON(&status)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid payload data"})
		return
	}

	// TODO: handle this after refactor
	// err = libp2p.ChangeDeviceStatus(status.IsAvailable)
	// if err != nil {
	// 	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
	// 	return
	// }
	c.AbortWithStatusJSON(500, gin.H{"error": "change device status not implemented"})
	// END

	var msg string
	if status.IsAvailable {
		msg = "Device status successfully changed to online"
	} else {
		msg = "Device status successfully changed to offline"
	}
	c.JSON(200, gin.H{"message": msg})
}
