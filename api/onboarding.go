package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/types"
)

// // OnboardingHandler is a controller for /onboarding endpoint functionalities
// type OnboardingHandler struct {
// 	service *onboarding.Onboarding
// }

// // NewOnboardingHandler is a constructor for OnboardingHandler
// func NewOnboardingHandler(s *onboarding.Onboarding) OnboardingHandler {
// 	return OnboardingHandler{service: s}
// }

// ProvisionedCapacity      godoc
//
//	@Summary		Returns provisioned capacity on host.
//	@Description	Get total memory capacity in MB and CPU capacity in MHz.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object}	types.Provisioned
//	@Router			/onboarding/provisioned [get]
func (rs *RESTServer) ProvisionedCapacity(c *gin.Context) {
	machineResources, err := rs.config.Resource.SystemSpecs().GetMachineResources()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, machineResources)
}

// CreatePaymentAddress      godoc
//
//	@Summary		Create a new payment address.
//	@Description	Create a payment address from public key. Return payment address and private key.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object}	types.BlockchainAddressPrivKey
//	@Router			/onboarding/address/new [get]
func (rs RESTServer) CreatePaymentAddress(c *gin.Context) {
	wallet := c.DefaultQuery("blockchain", "cardano")
	pair, err := onboarding.CreatePaymentAddress(wallet)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pair)
}

// Onboard      godoc
//
//	@Summary		Runs the onboarding process.
//	@Description	Onboard runs onboarding script given the amount of resources to onboard.
//	@Tags			onboarding
//	@Produce		json
//	@Param			capacity	body		types.CapacityForNunet	true	"Capacity for NuNet"
//	@Success		201			{object}	types.OnboardingConfig
//	@Failure		400			{object}	object	"invalid request data"
//	@Failure		500			{object}	object	"could not check if config directory exists"
//	@Failure		500			{object}	object	"config directory does not exist"
//	@Failure		500			{object}	object	"could not validate payment address"
//	@Failure		500			{object}	object	"could not validate capacity data"
//	@Failure		500			{object}	object	"cardano node requires 10000MB of RAM and 6000MHz CPU"
//	@Failure		500			{object}	object	"invalid channel data, channel does not exist"
//	@Failure		500			{object}	object	"unable to create available resources table"
//	@Failure		500			{object}	object	"unable to update available resources table"
//	@Failure		500			{object}	object	"could not calculate free resources and update database"
//	@Failure		500			{object}	object	"could not register and run new node"
//	@Router			/onboarding/onboard [post]
func (rs *RESTServer) Onboard(c *gin.Context) {
	capacity := types.CapacityForNunet{
		ServerMode:  true,
		IsAvailable: true,
	}

	if err := c.BindJSON(&capacity); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request data"})
		return
	}

	oConfig, err := rs.config.Onboarding.Onboard(c.Request.Context(), capacity)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, oConfig)
}

// Offboard      godoc
//
//	@Summary		Runs the offboarding process.
//	@Description	Offboard runs offboarding process to remove the machine from the NuNet network.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200		{string}	string	"device successfully offboarded"
//	@Param			force	query		string	false	"force offboarding"
//	@Failure		400		{object}	object	"invalid query data"
//	@Failure		500		{object}	object	"could not retrieve onboard status"
//	@Failure		500		{object}	object	"machine is not onboarded"
//	@Failure		500		{object}	object	"unable to shutdown node"
//	@Failure		500		{object}	object	"unable to delete available resources on database"
//	@Failure		500		{object}	object	"could not remove payment address"
//	@Router			/onboarding/offboard [post]
func (rs RESTServer) Offboard(c *gin.Context) {
	query := c.DefaultQuery("force", "false")
	force, err := strconv.ParseBool(query)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid query data"})
		return
	}

	err = rs.config.Onboarding.Offboard(c.Request.Context(), force)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device successfully offboarded"})
}

// Status      godoc
//
//	@Summary		Returns whether device is onboarded or not.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{boolean}
//	@Router			/onboarding/status [get]
func (rs RESTServer) Status(c *gin.Context) {
	status, err := rs.config.Onboarding.IsOnboarded(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"onboarded": status})
}

// Info      godoc
//
//	@Summary		Returns additional information about onboarded device.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object} 	types.OnboardingConfig
//	@Router			/onboarding/info [get]
func (rs RESTServer) Info(c *gin.Context) {
	info, err := rs.config.Onboarding.Info(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"info": info})
}

// ResourceConfig        godoc
//
//	@Summary	changes the amount of resources of onboarded device .
//	@Tags		onboarding
//	@Produce	json
//	@Success	200	{object}	types.OnboardingConfig
//	@Router		/onboarding/resource-config [post]
func (rs RESTServer) ResourceConfig(c *gin.Context) {
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "request body is empty"})
		return
	}

	var capacity types.CapacityForNunet
	err := c.BindJSON(&capacity)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request data"})
		return
	}

	oConfig, err := rs.config.Onboarding.ResourceConfig(c.Request.Context(), capacity)
	if err != nil {
		switch err {
		case onboarding.ErrMachineNotOnboarded:
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, oConfig)
}
