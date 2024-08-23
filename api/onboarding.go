package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
)

// OnboardingHandler is a controller for /onboarding endpoint functionalities
type OnboardingHandler struct {
	service *onboarding.Onboarding
}

// NewOnboardingHandler is a constructor for OnboardingHandler
func NewOnboardingHandler(s *onboarding.Onboarding) OnboardingHandler {
	return OnboardingHandler{service: s}
}

// ProvisionedCapacity      godoc
//
//	@Summary		Returns provisioned capacity on host.
//	@Description	Get total memory capacity in MB and CPU capacity in MHz.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object}	types.Provisioned
//	@Router			/onboarding/provisioned [get]
func (h OnboardingHandler) ProvisionedCapacity(c *gin.Context) {
	provisionedResources, err := resources.ManagerInstance.SystemSpecs().GetProvisionedResources()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, provisionedResources)
}

// CreatePaymentAddressHandler      godoc
//
//	@Summary		Create a new payment address.
//	@Description	Create a payment address from public key. Return payment address and private key.
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object}	types.BlockchainAddressPrivKey
//	@Router			/onboarding/address/new [get]
func (h OnboardingHandler) CreatePaymentAddress(c *gin.Context) {
	wallet := c.DefaultQuery("blockchain", "cardano")
	pair, err := onboarding.CreatePaymentAddress(wallet)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pair)
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
func (h OnboardingHandler) Onboard(c *gin.Context) {
	capacity := types.CapacityForNunet{
		ServerMode:  true,
		IsAvailable: true,
	}

	if err := c.BindJSON(&capacity); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request data"})
		return
	}

	oConfig, err := h.service.Onboard(c.Request.Context(), capacity)
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
func (h OnboardingHandler) Offboard(c *gin.Context) {
	query := c.DefaultQuery("force", "false")
	force, err := strconv.ParseBool(query)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid query data"})
		return
	}

	err = h.service.Offboard(c.Request.Context(), force)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device successfully offboarded"})
}

// OnboardStatus      godoc
//
//	@Summary		Onboarding status and additional info.
//	@Description	Returns json with 5 parameters: onboarded, error, machine_uuid, work_dir, database_path.
//	@Description	`onboarded` is true if the device is onboarded, false otherwise.
//	@Description	`error` is the error message if any related to onboarding status check
//	@Description	`machine_uuid` is the UUID of the machine
//	@Description	`work_dir` is the path to DMS's working directory
//	@Description	`database_path` is the path to nunet.db only if it exists
//	@Tags			onboarding
//	@Produce		json
//	@Success		200	{object}	types.OnboardingStatus
//	@Router			/onboarding/status [get]
func (h OnboardingHandler) OnboardStatus(c *gin.Context) {
	status, err := h.service.Status(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

// ResourceConfig        godoc
//
//	@Summary	changes the amount of resources of onboarded device .
//	@Tags		onboarding
//	@Produce	json
//	@Success	200	{object}	types.OnboardingConfig
//	@Router		/onboarding/resource-config [post]
func (h OnboardingHandler) ResourceConfig(c *gin.Context) {
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

	oConfig, err := h.service.ResourceConfig(c.Request.Context(), capacity)
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
