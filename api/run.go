package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gitlab.com/nunet/device-management-service/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type checkpoint struct {
	CheckpointDir string `json:"checkpoint_dir"`
	FilenamePath  string `json:"filename_path"`
	LastModified  int64  `json:"last_modified"`
}

// RequestServiceHandler  godoc
//
//	@Summary		RequestServiceHandler receives parameters from the SP client and returns blockchain realted data to client.
//	@Description	RequestServiceHandler receives the parameters from the SP client, searches the DHT for non-busy, available devices with appropriate metadata. Then informs parameters related to blockchain to client to run a service on NuNet.
//	@Tags			run
//	@Produce		json
//	@Param			deployment_request	body		models.DeploymentRequest	true	"Deployment Request"
//	@Success		200					{object}	machines.FundingRespToSPD
//	@Failure		400					{object}	object	"invalid payload data"
//	@Failure		500					{object}	object	"a service is already running; only 1 service is supported at the moment"
//	@Failure		500					{object}	object	"unable to obtain public key"
//	@Failure		500					{object}	object	"could not decode the peer id"
//	@Failure		500					{object}	object	"nunet estimation price is greater than client price"
//	@Failure		500					{object}	object	"targeted peer is not within host DHT"
//	@Failure		500					{object}	object	"no peers found with matched specs"
//	@Failure		500					{object}	object	"oracle connection error"
//	@Failure		500					{object}	object	"cannot write to database"
//	@Router			/run/request-service [post]
func RequestServiceHandler(c *gin.Context) {
	// reqCtx := c.Request.Context()

	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(attribute.String("URL", "/run/request-service"))

	// TODO: Uncomment after refactor.
	// kLogger.Info("Handle request service", span)
	// END

	var depReq models.DeploymentRequest
	err := c.BindJSON(&depReq)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid payload data"})
		return
	}
	// resp, err := machines.RequestService(reqCtx, depReq)
	// if err != nil {
	// 	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
	// 	return
	// }
	c.AbortWithStatusJSON(500, gin.H{"error": "RequestServiceHandler not implemented"})

	// c.JSON(200, resp)
}

// DeploymentRequestHandler  godoc
//
//	@Summary		Websocket endpoint responsible for sending deployment request and receiving deployment response.
//	@Description	Loads deployment request from the DB after a successful blockchain transaction has been made and passes it to compute provider.
//	@Tags			run
//	@Success		200	{string}	string
//	@Failure		500	{object}	object	"failed to set websocket upgrade"
//	@Router			/run/deploy [get]
func DeploymentRequestHandler(c *gin.Context) {
	reqCtx := c.Request.Context()

	span := trace.SpanFromContext(reqCtx)
	span.SetAttributes(attribute.String("URL", "/run/deploy"))

	// err := machines.DeploymentRequest(c, reqCtx, c.Writer, c.Request)
	// if err != nil {
	// 	c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
	// 	return
	// }
	c.AbortWithStatusJSON(500, gin.H{"error": "DeploymentRequestHandler not implemented"})
	// TODO: Original func did not return a success response. Should we return it?
}

// ListCheckpointHandler  godoc
//
//	@Summary		Returns a list of absolute path to checkpoint files. If no files are found in checkpoint directory, an empty array is returned.
//	@Description	ListCheckpointHandler scans data_dir/received_checkpoints and lists all the tar.gz files which can be used to resume a job. Returns a list of objects with absolute path and last modified date. If no files are found in checkpoint directory, an empty array is returned.
//	@Tags			run
//	@Success		200	{object}	[]checkpoint
//	@Failure		500	{object}	object	"failed to get checkpoint list"
//	@Router			/run/checkpoints [get]
func ListCheckpointHandler(c *gin.Context) {
	// TODO: Remove this section after refactor.
	checkpoints := checkpoint{}
	err := errors.New("non existant import path")
	// END

	// checkpoints, err := libp2p.ListCheckpoints()
	if err != nil {
		// c.AbortWithStatusJSON(500, gin.H{"error": "failed to get checkpoint list"})
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, checkpoints)
}
