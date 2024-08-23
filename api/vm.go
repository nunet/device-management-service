package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

type CustomVM struct {
	KernelImagePath string `json:"kernel_image_path"`
	FilesystemPath  string `json:"filesystem_path"`
	VCPUCount       int32  `json:"vcpu_count"`
	MemSizeMib      int    `json:"mem_size_mib"`
	TapDevice       string `json:"tap_device"`
}

type DefaultVM struct {
	KernelImagePath string `json:"kernel_image_path"`
	FilesystemPath  string `json:"filesystem_path"`
	PublicKey       string `json:"public_key"`
	NodeID          string `json:"node_id"`
}

// VMHandler is a controller for /vm endpoint functionalities
// TODO: Create a service type for these functionalities
// and embed inside the handler
type VMHandler struct{}

// StartCustom godoc
//
// @Summary		Start a VM with custom configuration.
// @Description	This endpoint is an abstraction of all primitive endpoints. When invokend, it calls all primitive endpoints in a sequence.
// @Tags			vm
// @Produce		json
// @Param			body	body		firecracker.CustomVM	true	"body"
// @Success		200		{object}	string					"VM started successfully."
// @Failure		400		{object}	string					"invalid request body"
// @Failure		500		{object}	string					"could not create database table"
// @Failure		500		{object}	string					"could not initialize virtual machine"
// @Failure		500		{object}	string					"failed to configure drives"
// @Failure		500		{object}	string					"failed to configure machine config"
// @Failure		500		{object}	string					"failed to configure network-interfaces"
// @Failure		500		{object}	string					"failed to setup MMDS"
// @Failure		500		{object}	string					"failed to pass MMDS message"
// @Failure		500		{object}	string					"unable to start virtual machine"
// @Router			/vm/start-custom [post]
func (h *VMHandler) StartCustom(c *gin.Context) {
	reqCtx := c.Request.Context()
	span := trace.SpanFromContext(reqCtx)
	span.SetAttributes(attribute.String("URL", "/vm/start-custom"))

	var body CustomVM
	err := c.BindJSON(&body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	fe := firecracker.NewFirecrackerEngineBuilder(body.FilesystemPath).
		WithKernelImage(body.KernelImagePath).
		Build()

	fer := &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: "test_execution",
		EngineSpec:  fe,
		Resources: &types.ExecutionResources{
			CPU:    types.CPU{Cores: uint32(body.VCPUCount)},
			Memory: types.RAM{Size: int64(body.MemSizeMib)},
		},
	}

	fc, err := firecracker.NewExecutor(c.Request.Context(), "manual-start-custom")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = fc.Start(c.Request.Context(), fer)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "VM started successfully"})
}

// StartDefault godoc
//
//	@Summary		Start a VM with default configuration.
//	@Description	Kernel file and filesystem file needs to be passed in body. This endpoint is an abstraction of all primitive endpoints.
//	@Tags			vm
//	@Produce		json
//	@Param			body	body		firecracker.DefaultVM	true	"body"
//	@Success		200		{object}	string					"VM started successfully."
//	@Failure		400		{object}	string					"invalid request body"
//	@Failure		500		{object}	string					"could not initialize virtual machine"
//	@Failure		500		{object}	string					"failed to confiugre boot source"
//	@Failure		500		{object}	string					"failed to configure drives"
//	@Failure		500		{object}	string					"failed to configure machineConfig"
//	@Failure		500		{object}	string					"failed to configure network-interfaces"
//	@Failure		500		{object}	string					"failed to setup MMDS"
//	@Failure		500		{object}	string					"failed to pass MMDS message"
//	@Failure		500		{object}	string					"unable to start virtual machine"
//	@Router			/vm/start-default [post]
func (h *VMHandler) StartDefault(c *gin.Context) {
	reqCtx := c.Request.Context()
	span := trace.SpanFromContext(reqCtx)
	span.SetAttributes(attribute.String("URL", "/vm/start-default"))

	var body DefaultVM
	err := c.BindJSON(&body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	fe := firecracker.NewFirecrackerEngineBuilder(body.FilesystemPath).
		WithKernelImage(body.KernelImagePath).
		Build()

	fer := &types.ExecutionRequest{
		JobID:       "test_job",
		ExecutionID: "test_execution",
		EngineSpec:  fe,
		Resources: &types.ExecutionResources{
			CPU:    types.CPU{Cores: 1},
			Memory: types.RAM{Size: 1024},
		},
	}

	fc, err := firecracker.NewExecutor(c.Request.Context(), "manual-start-default")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = fc.Start(c.Request.Context(), fer)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "VM started successfully"})
}
