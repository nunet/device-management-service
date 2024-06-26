// To run a single test use "-run=^TestSecurity\$ -testify.m TestTxHashValidation"

package main

// import (
// 	"bufio"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"math"
// 	"os"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/libp2p/go-libp2p/core/crypto"
// 	"github.com/libp2p/go-libp2p/core/peer"
// 	"github.com/stretchr/testify/suite"
// 	"gitlab.com/nunet/device-management-service/db"
// 	"gitlab.com/nunet/device-management-service/integrations/oracle"
// 	"gitlab.com/nunet/device-management-service/internal/config"
// 	"gitlab.com/nunet/device-management-service/internal/heartbeat"
// 	"gitlab.com/nunet/device-management-service/internal/messaging"
// 	"gitlab.com/nunet/device-management-service/internal/tracing"
// 	"gitlab.com/nunet/device-management-service/models"
// 	"gitlab.com/nunet/device-management-service/network/libp2p"
// 	"gitlab.com/nunet/device-management-service/utils"
// 	"gitlab.com/nunet/device-management-service/utils/cardano"
// )

// // This is a transaction hash that exists on preprpod
// const OldValidTransactionHash = "723fb1e2260c8f7c31b5760177c365917fcb39291925ce8b5c897c51d0de8fe7"

// // Address of the SP, currently same as CP
// const RequesterAddress = "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk"
// const TesterKeyHash = "906b4d1d751bd204e60083bec876df93e4ff241007291fe7486993cb"
// const SPKeyHash = "906b4d1d751bd204e60083bec876df93e4ff241007291fe7486993cb"
// const CPKeyHash = "bd22ee8fe84c5286f6eae37ac37559d4e6743aa64520e686fc6d8e1b"
// const TxConfirmationTimeoutMinutes = 5

// type TestHarness struct {
// 	suite.Suite
// 	sync.WaitGroup
// 	cleanup func(context.Context) error
// }

// func TestSecurity(t *testing.T) {
// 	s := new(TestHarness)
// 	suite.Run(t, s)
// }

// func (s *TestHarness) SetupSuite() {
// 	SetupDMSTestingConfiguration("target", 0)
// 	OnboardTestComputeProvider()
// 	RunTestComputeProvider()

// 	s.cleanup = tracing.InitTracer()
// 	s.WaitGroup.Add(1)
// }

// func (s *TestHarness) TearDownSuite() {
// 	s.WaitGroup.Done()
// 	s.cleanup(context.Background())
// 	os.RemoveAll("/tmp/nunet-target")
// 	os.RemoveAll("/tmp/nunet-client")
// }

// func (spClient *SPTestClient) DefaultDeploymentRequest(tx_hash string) models.DeploymentRequest {
// 	var req models.DeploymentRequest
// 	// Hash of a valid job that has a valid datum and payment, but that doesn't list this CP DMS's address as the chosen CP
// 	req.TxHash = tx_hash
// 	req.RequesterWalletAddress = RequesterAddress
// 	req.MaxNtx = 2
// 	req.Blockchain = "Cardano"
// 	req.ServiceType = "ml-training-cpu"
// 	req.Timestamp = time.Now()
// 	req.Params.MachineType = "cpu"

// 	// Simple Model URL
// 	req.Params.ModelURL = basicTensorflowModelURL

// 	// Valid cpu image ID
// 	req.Params.ImageID = "registry.gitlab.com/nunet/ml-on-gpu/ml-on-cpu-service/develop/ml-on-cpu"

// 	cpId := GetTestComputeProviderID()

// 	computeProviderPubKey, err := libp2p.GetP2P().Host.Peerstore().PubKey(cpId).Raw()
// 	spClient.s.Nil(err, "Failed to obtain compute provider public key")

// 	// NOTE (divam): currently the public key is added to JSON without the typical base64 encoding
// 	req.Params.RemoteNodeID = cpId.String()
// 	req.Params.RemotePublicKey = string(computeProviderPubKey)
// 	req.Params.LocalNodeID = spClient.selfID.String()
// 	selfPubKey, _ := spClient.selfPublicKey.Raw()
// 	req.Params.LocalPublicKey = string(selfPubKey)

// 	// Low complexity and constraints
// 	req.Constraints.Complexity = "Low"
// 	req.Constraints.CPU = 1500
// 	req.Constraints.RAM = 2000
// 	req.Constraints.Vram = 2000
// 	req.Constraints.Power = 170
// 	req.Constraints.Time = 1

// 	oracleResp, err := oracle.FundContractRequest(&oracle.FundingRequest{
// 		ServiceProviderAddr: req.RequesterWalletAddress,
// 		// TODO: obtain from CP metadata
// 		ComputeProviderAddr: "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk",
// 		EstimatedPrice:      int64(req.MaxNtx),
// 	})
// 	spClient.s.Nil(err, "Failed to obtain oracleResp")

// 	req.MetadataHash = oracleResp.MetadataHash
// 	req.WithdrawHash = oracleResp.WithdrawHash
// 	req.RefundHash = oracleResp.RefundHash
// 	req.Distribute_50Hash = oracleResp.Distribute_50Hash
// 	req.Distribute_75Hash = oracleResp.Distribute_75Hash

// 	return req
// }

// // Convenience function to check for job failure and if the job runs respond with a custom error message.
// func (spClient *SPTestClient) AssertJobFail(job_ran_error string) {
// 	s := spClient.s

// 	firstUpdate, err := spClient.GetNextDeploymentUpdate()
// 	s.Nil(err, "Failed to get first deployment update")
// 	s.NotEqual(libp2p.MsgJobStatus, firstUpdate.MsgType, job_ran_error)

// 	// If we have received a message a job is running, lets also confirm that the DeploymentRequest is successful as well
// 	if firstUpdate.MsgType == libp2p.MsgJobStatus {
// 		secondUpdate, err := spClient.GetNextDeploymentUpdate()
// 		s.Equal(libp2p.MsgDepResp, secondUpdate.MsgType, "We didn't receive a DeploymentResponse for our DeploymentRequest")
// 		s.Nil(err, "Failed to get second deployment update")
// 		s.Equal(false, secondUpdate.Response.Success, job_ran_error)

// 		if secondUpdate.Response.Success {
// 			// Wait for CP to finish the current job, as only one job can run at a time
// 			// This is to prevent test failure due to "depReq already in progress. Refusing to accept."
// 			for {
// 				update, err := spClient.GetNextDeploymentUpdate()
// 				if err != nil {
// 					s.Nil(err, "Failed to get deployment update")
// 					break
// 				}
// 				if update.MsgType == libp2p.MsgLogStdout || update.MsgType == libp2p.MsgLogStderr {
// 					continue
// 				}
// 				// Final confirmation that the job finished
// 				s.Equal(libp2p.MsgJobStatus, update.MsgType, "Expected Job Status update")
// 				break
// 			}
// 		}
// 	}
// }

// // Convenience  function like AssertJobFail but just runs through making sure we have status updates
// func (spClient *SPTestClient) RunJob() {
// 	s := spClient.s

// 	firstUpdate, err := spClient.GetNextDeploymentUpdate()
// 	s.Nil(err, "Failed to get first deployment update for success")

// 	// If we have received a message a job is running, lets also confirm that the DeploymentRequest is successful as well
// 	if firstUpdate.MsgType == libp2p.MsgJobStatus {
// 		secondUpdate, err := spClient.GetNextDeploymentUpdate()
// 		s.Equal(libp2p.MsgDepResp, secondUpdate.MsgType, "We didn't receive a DeploymentResponse for our DeploymentRequest")
// 		s.Nil(err, "Failed to get second deployment update for success")

// 		if secondUpdate.Response.Success {
// 			// Wait for CP to finish the current job, as only one job can run at a time
// 			// This is to prevent test failure due to "depReq already in progress. Refusing to accept."
// 			for {
// 				update, err := spClient.GetNextDeploymentUpdate()
// 				if err != nil {
// 					s.Nil(err, "Failed to get deployment update")
// 					break
// 				}
// 				if update.MsgType == libp2p.MsgLogStdout || update.MsgType == libp2p.MsgLogStderr {
// 					continue
// 				}
// 				// Final confirmation that the job finished
// 				s.Equal(libp2p.MsgJobStatus, update.MsgType, "Expected Job Status update")
// 				break
// 			}
// 		}
// 	}
// }

// // Test that the CP DMS will run a job with a valid tx hash
// func (s *TestHarness) TestDMSRunsValidJob() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	const ntxAmount = 2

// 	tx_hash, err := cardano.PayToScript(ntxAmount, TesterKeyHash, TesterKeyHash)
// 	if err != nil {
// 		s.Nil(err, "Failed to pay to script")
// 		return
// 	}

// 	req := spClient.DefaultDeploymentRequest(tx_hash)

// 	spClient.SendDeploymentRequest(req)

// 	// Confirm that the job ran successfully
// 	spClient.RunJob()
// }

// // Test that the CP DMS will not run a job with an invalid tx hash
// func (s *TestHarness) TestTxHashValidation() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest("invalidtxhash")

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent an invalid tx-hash but the CP confirmed deployment success")
// }

// // Test if the the CP DMS will run a undervalued job
// func (s *TestHarness) TestTxUndervaluation() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	// Create a request with very high constraints but with the minimum NTX
// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.MaxNtx = 1
// 	req.Constraints.CPU = 1000000
// 	req.Constraints.RAM = 2000000
// 	req.Constraints.Vram = 2000000
// 	req.Constraints.Power = 20000000
// 	req.Constraints.Time = 10000000

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("The CP DMS ran the job even though the requirements are too high for the payment")
// }

// /*func (s *TestHarness) TestInvalidResume() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.ResumeJob.Resume = true
// 	req.Params.ResumeJob.ProgressFile = "null"

// 	spClient.SendDeploymentRequest(req)
// 	spClient.AssertJobFail("Attempted to start invalid resume")
// }*/

// func (s *TestHarness) TestMultipleRequestsSameTX() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)

// 	reqnew := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	reqnew.MaxNtx = 5

// 	spClient.SendDeploymentRequest(req)
// 	spClient.RunJob()

// 	spClient.SendDeploymentRequest(reqnew)
// 	spClient.AssertJobFail("Attempted to start a job with a previously used tx_hash")
// }

// // Test that the CP DMS will not run a job with a valid tx hash that does not have the correct amount of NTX
// func (s *TestHarness) TestTxNTXValidation() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.MaxNtx = 1000000

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a valid transaction but decalred a higher payout to the DMS and the job ran success")
// }

// // Test that the CP DMS will only run the job when the Params specifying a correct LocalPublicKey
// func (s *TestHarness) TestValidSPPublicKey() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.LocalPublicKey = "invalid-local-key"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid LocalPublicKey")
// }

// // Test that the CP DMS will only run the job when the Params specifying a correct LocalNodeID
// func (s *TestHarness) TestValidSPNodeId() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.LocalNodeID = "invalid-local-id"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid LocalNodeID")
// }

// // Test that the CP DMS will only run the job when the Params specifying a correct RemotePublicKey
// func (s *TestHarness) TestValidCPPublicKey() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.RemotePublicKey = "invalid-remote-key"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid RemotePublicKey")
// }

// // Test that the CP DMS will only run the job when the Params specifying a correct RemoteNodeID
// func (s *TestHarness) TestValidCPNodeId() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.RemoteNodeID = "invalid-remote-id"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid RemoteNodeID")
// }

// // Test that the CP DMS will only run a valid docker image
// func (s *TestHarness) TestValidImageID() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.ImageID = "registry.hub.docker.com/library/busybox"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid ImageID")
// }

// // Test that the CP DMS will only run a valid paylaod
// func (s *TestHarness) TestValidModelURL() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	// contains malicious code doing process fork
// 	req.Params.ModelURL = "https://gist.githubusercontent.com/cidkidnix/1a9245b464fc8d05e95778dc5fb6255c/raw/f87c196dd214994eb632c17d50837460542ccb4e/gistfile1.py"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid ModelURL")
// }

// // Test that the CP DMS will only run a valid service type
// func (s *TestHarness) TestValidServiceType() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")
// 	defer spClient.ShutdownSPClient()

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.ServiceType = "invalid"

// 	spClient.SendDeploymentRequest(req)

// 	spClient.AssertJobFail("Malicious SP sent a DeploymentRequest with invalid ServiceType")
// }

// // Test that the SP cannot request refund while the job is executing
// func (s *TestHarness) TestSPCannotRefundWhileJobIsExecuting() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.ModelURL = oneMinSleepModelURL

// 	spClient.SendDeploymentRequest(req)

// 	firstUpdate, err := spClient.GetNextDeploymentUpdate()
// 	s.Nil(err, "Failed to get first deployment update")
// 	s.Equal(libp2p.MsgJobStatus, firstUpdate.MsgType, "Failed to deploy job")

// 	// Confirm that the DeploymentRequest is successful
// 	if firstUpdate.MsgType == libp2p.MsgJobStatus {
// 		secondUpdate, err := spClient.GetNextDeploymentUpdate()
// 		s.Equal(libp2p.MsgDepResp, secondUpdate.MsgType, "We didn't receive a DeploymentResponse for our DeploymentRequest")
// 		s.Nil(err, "Failed to get second deployment update")

// 		// Now that the Job has been confirmed to be running, obtain signatures for refund
// 		// SP can obtain the signature based on parameters already known
// 		oracleResp := spClient.getSignaturesFromOracle(req, OracleRewardReqPayload{
// 			JobStatus:            "finished with errors",
// 			JobDuration:          int64(0),
// 			EstimatedJobDuration: int64(req.Constraints.Time),
// 			LogURL:               "https://log.nunet.io/api/v1/logbin/invalid/raw",
// 		})

// 		// The SP can in principle obtain a refund using the signature obtained from oracle
// 		// while waiting for the job to complete
// 		s.NotEqual("refund", oracleResp.RewardType, "Obtained a refund request for a running/valid job")
// 		s.NotEqual(128, len(oracleResp.SignatureDatum), "Obtained signature from oracle for a refund request for a running/valid job")

// 		for {
// 			update, err := spClient.GetNextDeploymentUpdate()
// 			s.Nil(err, "Failed to get deployment update")
// 			if update.MsgType == libp2p.MsgLogStdout || update.MsgType == libp2p.MsgLogStderr {
// 				continue
// 			}
// 			// Final confirmation that the job finished without errors and that SP obtains a valid LogURL from CP
// 			s.Equal(libp2p.MsgJobStatus, update.MsgType, "Expected Job Status update")
// 			s.Equal("finished without errors", update.Services.JobStatus, "Expected Job to have finished succesfully")
// 			s.NotEqual(0, len(update.Services.LogURL), "Expected LogURL to be non empty")
// 			break
// 		}
// 	}
// }

// // Test that the SP cannot request refund before the timeout of job, even if CP goes offline
// // Note: This is currently identical to TestSPCannotRefundWhileJobIsExecuting,
// // as currntly the SP does not rely on response from CP for executing a refund request
// func (s *TestHarness) TestSPCannotRefundBeforeTimeoutCPOffline() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.ModelURL = oneMinSleepModelURL

// 	spClient.SendDeploymentRequest(req)

// 	firstUpdate, err := spClient.GetNextDeploymentUpdate()
// 	s.Nil(err, "Failed to get first deployment update")
// 	s.Equal(libp2p.MsgJobStatus, firstUpdate.MsgType, "Failed to deploy job")

// 	// Confirm that the DeploymentRequest is successful
// 	if firstUpdate.MsgType == libp2p.MsgJobStatus {
// 		secondUpdate, err := spClient.GetNextDeploymentUpdate()
// 		s.Equal(libp2p.MsgDepResp, secondUpdate.MsgType, "We didn't receive a DeploymentResponse for our DeploymentRequest")
// 		s.Nil(err, "Failed to get second deployment update")

// 		// Now that the Job has been confirmed to be running, obtain signatures for refund
// 		// SP can obtain the signature based on parameters already known
// 		oracleResp := spClient.getSignaturesFromOracle(req, OracleRewardReqPayload{
// 			JobStatus:            "finished with errors",
// 			JobDuration:          int64(0),
// 			EstimatedJobDuration: int64(req.Constraints.Time),
// 			LogURL:               "https://log.nunet.io/api/v1/logbin/invalid/raw",
// 		})

// 		// The SP can in principle obtain a refund using the signature obtained from oracle
// 		// while waiting for the job to complete
// 		s.NotEqual("refund", oracleResp.RewardType, "Obtained a refund request for a running/valid job")
// 		s.NotEqual(128, len(oracleResp.SignatureDatum), "Obtained signature from oracle for a refund request for a running/valid job")

// 		for {
// 			update, err := spClient.GetNextDeploymentUpdate()
// 			s.Nil(err, "Failed to get deployment update")
// 			if update.MsgType == libp2p.MsgLogStdout || update.MsgType == libp2p.MsgLogStderr {
// 				continue
// 			}
// 			// Final confirmation that the job finished without errors and that SP obtains a valid LogURL from CP
// 			s.Equal(libp2p.MsgJobStatus, update.MsgType, "Expected Job Status update")
// 			s.Equal("finished without errors", update.Services.JobStatus, "Expected Job to have finished succesfully")
// 			s.NotEqual(0, len(update.Services.LogURL), "Expected LogURL to be non empty")
// 			break
// 		}
// 	}
// }

// // Test that SP cannot request refund after the job ran successfully
// func (s *TestHarness) TestSPCannotRefundAfterValidJob() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)
// 	req.Params.ModelURL = oneMinSleepModelURL

// 	spClient.SendDeploymentRequest(req)

// 	firstUpdate, err := spClient.GetNextDeploymentUpdate()
// 	s.Nil(err, "Failed to get first deployment update")
// 	s.Equal(libp2p.MsgJobStatus, firstUpdate.MsgType, "Failed to deploy job")

// 	// Confirm that the DeploymentRequest is successful
// 	if firstUpdate.MsgType == libp2p.MsgJobStatus {
// 		secondUpdate, err := spClient.GetNextDeploymentUpdate()
// 		s.Equal(libp2p.MsgDepResp, secondUpdate.MsgType, "We didn't receive a DeploymentResponse for our DeploymentRequest")
// 		s.Nil(err, "Failed to get second deployment update")

// 		// Wait for the job to finish successfully
// 		for {
// 			update, err := spClient.GetNextDeploymentUpdate()
// 			s.Nil(err, "Failed to get deployment update")
// 			if update.MsgType == libp2p.MsgLogStdout || update.MsgType == libp2p.MsgLogStderr {
// 				continue
// 			}
// 			// Final confirmation that the job finished without errors and that SP obtains a valid LogURL from CP
// 			s.Equal(libp2p.MsgJobStatus, update.MsgType, "Expected Job Status update")
// 			s.Equal("finished without errors", update.Services.JobStatus, "Expected Job to have finished succesfully")
// 			s.NotEqual(0, len(update.Services.LogURL), "Expected LogURL to be non empty")
// 			break
// 		}
// 	}

// 	// Now obtain signatures for refund, by pretending that the job finished with errors
// 	oracleResp := spClient.getSignaturesFromOracle(req, OracleRewardReqPayload{
// 		JobStatus:            "finished with errors",
// 		JobDuration:          int64(0),
// 		EstimatedJobDuration: int64(req.Constraints.Time),
// 		LogURL:               "https://log.nunet.io/api/v1/logbin/invalid/raw",
// 	})

// 	// Using these signatures the SP can do a refund request
// 	// At this point the CP DMS and SP DMS will likely compete for submitting the txs on chain.
// 	// It will be a race situation but the SP can win, as
// 	// the CP DMS does not send the withdraw tx to chain before sending the results.
// 	s.NotEqual("refund", oracleResp.RewardType, "Obtained a refund request for a running/valid job")
// 	s.NotEqual(128, len(oracleResp.SignatureDatum), "Obtained signature from oracle for a refund request for a running/valid job")
// }

// // Test that CP cannot request withdraw without running job successfully
// func (s *TestHarness) TestCPCannotWithdrawForInvalidResults() {
// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	req := spClient.DefaultDeploymentRequest(OldValidTransactionHash)

// 	// At the moment it is not necessary to even run the job to perform a withdraw
// 	// Therefore CP can respond that it is executing the job, while it does the withdraw
// 	// Now obtain signatures for withdraw, by pretending that the job finished successfully
// 	oracleResp := spClient.getSignaturesFromOracle(req, OracleRewardReqPayload{
// 		JobStatus:            "finished without errors",
// 		JobDuration:          int64(req.Constraints.Time),
// 		EstimatedJobDuration: int64(req.Constraints.Time),
// 		LogURL:               "https://log.nunet.io/api/v1/logbin/invalid/raw",
// 	})

// 	// Using these signatures the CP can do a withdraw request
// 	s.NotEqual("withdraw", oracleResp.RewardType, "Obtained a withdraw request without running the job")
// 	s.NotEqual(128, len(oracleResp.SignatureDatum), "Obtained signature from oracle for a withdraw request without running the job")
// }

// // Test if the CP can successfully withdraw
// func (s *TestHarness) TestValidCPWithdrawal() {
// 	hash, err := cardano.PayToScript(1, SPKeyHash, CPKeyHash)
// 	if err != nil {
// 		s.Fail("Failed to pay to script")
// 	}
// 	log.Printf("Transaction Hash %s", hash)
// 	err = cardano.WaitForTransaction(hash, TxConfirmationTimeoutMinutes)
// 	if err != nil {
// 		s.Fail("Failed to get tx confirmation")
// 	}
// 	err = cardano.SpendFromScript(hash, 0, cardano.Withdraw)
// 	if err != nil {
// 		s.Fail("Failed to spend from script")
// 	}
// }

// // Test if the SP can successfully refund
// func (s *TestHarness) TestValidSPRefund() {
// 	hash, err := cardano.PayToScript(1, SPKeyHash, CPKeyHash)
// 	if err != nil {
// 		s.Fail("Failed to pay to script")
// 	}
// 	log.Printf("Transaction Hash %s", hash)
// 	err = cardano.WaitForTransaction(hash, TxConfirmationTimeoutMinutes)
// 	if err != nil {
// 		s.Fail("Failed to get tx confirmation")
// 	}
// 	err = cardano.SpendFromScript(hash, 0, cardano.Refund)
// 	if err != nil {
// 		s.Fail("Failed to spend from script")
// 	}
// }

// // Test if the SP can successfully refund after timeout
// func (s *TestHarness) TestValidSPRefundAfterTimeout() {
// 	const ntxAmount = 2

// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	hash, err := cardano.PayToScript(ntxAmount, SPKeyHash, CPKeyHash)
// 	if err != nil {
// 		s.Fail("Failed to pay to script")
// 	}
// 	log.Printf("Transaction Hash %s", hash)
// 	err = cardano.WaitForTransaction(hash, TxConfirmationTimeoutMinutes)
// 	if err != nil {
// 		s.Fail("Failed to get tx confirmation")
// 	}

// 	req := spClient.DefaultDeploymentRequest(hash)
// 	req.MaxNtx = ntxAmount

// 	spClient.SendDeploymentRequest(req)

// 	// Time used in testing requests
// 	time.Sleep(time.Duration(req.Constraints.Time) * time.Minute)

// 	err = cardano.SpendFromScript(hash, 0, cardano.Refund)
// 	if err != nil {
// 		s.Fail("Failed to spend from script")
// 	}
// }

// // Test if the SP can successfully refund after timeout with the CP offline
// func (s *TestHarness) TestValidSPRefundWithOfflineCP() {
// 	const ntxAmount = 2

// 	spClient, err := CreateServiceProviderTestingClient(s)
// 	s.Nil(err, "Failed to create testing client")

// 	hash, err := cardano.PayToScript(ntxAmount, SPKeyHash, CPKeyHash)
// 	if err != nil {
// 		s.Fail("Failed to pay to script")
// 	}
// 	log.Printf("Transaction Hash %s", hash)
// 	err = cardano.WaitForTransaction(hash, TxConfirmationTimeoutMinutes)
// 	if err != nil {
// 		s.Fail("Failed to get tx confirmation")
// 	}

// 	req := spClient.DefaultDeploymentRequest(hash)
// 	req.MaxNtx = ntxAmount

// 	spClient.SendDeploymentRequest(req)

// 	// Time used in testing requests
// 	time.Sleep(time.Duration(req.Constraints.Time) * time.Minute)

// 	// Kill the node
// 	libp2p.ShutdownNode()

// 	err = cardano.SpendFromScript(hash, 0, cardano.Refund)
// 	if err != nil {
// 		s.Fail("Failed to spend from script")
// 	}
// }

// type OracleRewardReqPayload struct {
// 	JobStatus            string // whether job is running or exited; one of these 'running', 'finished without errors', 'finished with errors'
// 	JobDuration          int64  // job duration in minutes
// 	EstimatedJobDuration int64  // job duration in minutes
// 	LogURL               string
// }

// func (spClient *SPTestClient) getSignaturesFromOracle(req models.DeploymentRequest, payload OracleRewardReqPayload) (oracleResp *oracle.RewardResponse) {
// 	oracleResp, err := oracle.Oracle.WithdrawTokenRequest(&oracle.RewardRequest{
// 		JobStatus:            payload.JobStatus,
// 		JobDuration:          payload.JobDuration,
// 		EstimatedJobDuration: payload.EstimatedJobDuration,
// 		LogPath:              payload.LogURL,
// 		MetadataHash:         req.MetadataHash,
// 		WithdrawHash:         req.WithdrawHash,
// 		RefundHash:           req.RefundHash,
// 		Distribute_50Hash:    req.Distribute_50Hash,
// 		Distribute_75Hash:    req.Distribute_75Hash,
// 	})

// 	spClient.s.Nil(err, "Failed to obtain signatures from oracle")

// 	return oracleResp
// }

// func GenerateTestKeyPair() (crypto.PrivKey, crypto.PubKey, error) {
// 	priv, pub, err := crypto.GenerateKeyPair(
// 		crypto.Ed25519, // Ed25519 are nice short
// 		-1,             // No length required for Ed25519
// 	)

// 	return priv, pub, err
// }

// func SetupDMSTestingConfiguration(tempDirectoryName string, port int) {
// 	dmsTempDir := fmt.Sprintf("/tmp/nunet-%s", tempDirectoryName)
// 	os.Mkdir(dmsTempDir, 0755)
// 	ioutil.WriteFile(fmt.Sprintf("%s/metadataV2.json", dmsTempDir), []byte(testMetadata), 0644)
// 	config.LoadConfig()
// 	addrs := [2]string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port)}
// 	config.SetConfig("general.metadata_path", dmsTempDir)
// 	config.SetConfig("p2p.listen_address", addrs)
// }

// func OnboardTestComputeProvider() {
// 	db.ConnectDatabase()

// 	availableResources := models.AvailableResources{
// 		TotCpuHz:  int(2000000),
// 		CpuNo:     int(32),
// 		CpuHz:     2000,
// 		PriceCpu:  0,
// 		Ram:       int(21334),
// 		PriceRam:  0,
// 		Vcpu:      int(math.Floor((float64(2000000)) / 2000)),
// 		Disk:      0,
// 		PriceDisk: 0,
// 	}

// 	freeResources := models.FreeResources{
// 		ID:        1,
// 		TotCpuHz:  availableResources.TotCpuHz,
// 		PriceCpu:  0,
// 		Ram:       availableResources.Ram,
// 		PriceRam:  0,
// 		Vcpu:      0,
// 		Disk:      0,
// 		PriceDisk: 0,
// 	}

// 	db.DB.Create(&availableResources)
// 	db.DB.Create(&freeResources)
// }

// func RunTestComputeProvider() error {
// 	// Run deployment worker so that deployment requests are handled
// 	go messaging.DeploymentWorker()

// 	// Create a new key pair for Node Id Generation
// 	pair, _, err := GenerateTestKeyPair()

// 	runAsServer := true
// 	available := true
// 	libp2p.RunNode(pair, runAsServer, available)

// 	if libp2p.GetP2P().Host != nil {
// 		heartbeat.CheckToken(libp2p.GetP2P().Host.ID().String(), utils.GetChannelName())
// 	}
// 	return err
// }

// // Interface for SP to communicate with testing CP
// type SPTestClient struct {
// 	reader        *bufio.Reader
// 	writer        *bufio.Writer
// 	s             *TestHarness
// 	selfID        peer.ID
// 	selfPublicKey crypto.PubKey
// 	p2p           libp2p.DMSp2p
// }

// // Convenience type for DeploymentUpdate with unmarshalled data
// type CPUpdate struct {
// 	MsgType  string
// 	Response models.DeploymentResponse
// 	Services models.Services
// }

// func CreateServiceProviderTestingClient(s *TestHarness) (SPTestClient, error) {
// 	ctx := context.Background()

// 	SetupDMSTestingConfiguration("client", 0)

// 	pair, selfPublicKey, err := GenerateTestKeyPair()
// 	host, dht, err := libp2p.NewHost(ctx, pair, false)
// 	p2p := *libp2p.DMSp2pInit(host, dht)
// 	selfID := p2p.Host.ID()
// 	err = p2p.BootstrapNode(ctx)

// 	// After bootstrap wait a moment for the peer tables to initialize
// 	time.Sleep(2 * time.Second)

// 	cpID := GetTestComputeProviderID()

// 	stream, err := host.NewStream(context.Background(), cpID, libp2p.DepReqProtocolID)

// 	reader := bufio.NewReader(stream)
// 	writer := bufio.NewWriter(stream)
// 	return SPTestClient{reader, writer, s, selfID, selfPublicKey, p2p}, err
// }

// func (client *SPTestClient) ShutdownSPClient() error {
// 	for _, node := range client.p2p.Host.Peerstore().Peers() {
// 		client.p2p.Host.Network().ClosePeer(node)
// 		client.p2p.Host.Peerstore().Put(node, "peer_info", nil)
// 		client.p2p.Host.Peerstore().ClearAddrs(node)
// 		client.p2p.Host.Peerstore().RemovePeer(node)
// 	}

// 	err := client.p2p.Host.Close()
// 	if err != nil {
// 		return err
// 	}
// 	err = client.p2p.DHT.Close()
// 	if err != nil {
// 		return err
// 	}

// 	client.p2p.Host = nil
// 	client.p2p.DHT = nil

// 	// Ensure the bootstrapnode does remove SPTestClient before starting another test
// 	time.Sleep(2 * time.Second)
// 	return nil
// }

// func GetTestComputeProviderID() peer.ID {
// 	p2p := libp2p.GetP2P()
// 	return p2p.Host.ID()
// }

// func (client *SPTestClient) SendDeploymentRequest(request models.DeploymentRequest) error {
// 	msg, json_err := json.Marshal(request)
// 	if json_err != nil {
// 		return json_err
// 	}

// 	_, write_err := client.writer.WriteString(fmt.Sprintf("%s\n", msg))
// 	if write_err != nil {
// 		return write_err
// 	}

// 	flush_err := client.writer.Flush()
// 	if flush_err != nil {
// 		return flush_err
// 	}

// 	return nil
// }

// type ReadResult struct {
// 	Message string
// 	Error   error
// }

// func (client *SPTestClient) GetNextDeploymentUpdate() (CPUpdate, error) {
// 	var update CPUpdate

// 	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
// 	defer cancel()

// 	ch := make(chan ReadResult)

// 	go func() {
// 		msg, err := client.reader.ReadString('\n')
// 		ch <- ReadResult{Message: msg, Error: err}
// 	}()

// 	msg := ""
// 	select {
// 	case <-ctx.Done():
// 		return update, errors.New(fmt.Sprintf("GetNextDeploymentUpdate timedout"))
// 	case result := <-ch:
// 		if result.Error != nil {
// 			return update, result.Error
// 		}
// 		msg = result.Message
// 	}

// 	depUpdate := models.DeploymentUpdate{}
// 	err := json.Unmarshal([]byte(msg), &depUpdate)
// 	if err != nil {
// 		return update, err
// 	}

// 	update = CPUpdate{
// 		MsgType: depUpdate.MsgType,
// 	}

// 	switch depUpdate.MsgType {
// 	case libp2p.MsgDepResp:
// 		err = json.Unmarshal([]byte(depUpdate.Msg), &update.Response)
// 		if err != nil {
// 			return update, err
// 		}
// 	case libp2p.MsgJobStatus:
// 		json.Unmarshal([]byte(depUpdate.Msg), &update.Services)
// 		if err != nil {
// 			return update, err
// 		}
// 	case libp2p.MsgLogStdout:
// 	case libp2p.MsgLogStderr:
// 	default:
// 		return update, errors.New(fmt.Sprintf("Invalid Message Type %s", depUpdate.MsgType))
// 	}

// 	return update, nil
// }

// // Generated metadata for testing
// var testMetadata string = `
// {
//  "update_timestamp": 1698332902,
//  "resource": {
//   "memory_max": 31674,
//   "total_core": 16,
//   "cpu_max": 67198
//  },
//  "available": {
//   "cpu": 42942,
//   "memory": 10340
//  },
//  "reserved": {
//   "cpu": 24256,
//   "memory": 21334
//  },
//  "network": "nunet-team",
//  "public_key": "addr_test1vzgxkngaw5dayp8xqzpmajrkm7f7fleyzqrjj8l8fp5e8jcc2p2dk",
//  "allow_cardano": true
// }`

// // Prints 'GPU found' / 'No GPU found'
// var basicTensorflowModelURL = "https://gist.githubusercontent.com/luigy/d63eec5cb33d9f789969fafe04ee3ae9/raw/c9722361c24e7520e5ebc084f94358fc0858753e/tesorflow.py"

// // Prints 'GPU found' / 'No GPU found' and sleep for 1 min
// var oneMinSleepModelURL = "https://gist.githubusercontent.com/dfordivam/10fa4b73f1d51cfc0d94aea844634bf7/raw/01490c73a7e988a11484b0ed42946cb3422b570a/one-min-sleep.py"

// // Prints 'GPU found' / throws error on CPU
// var gpuOnlyModelURL = "https://gist.githubusercontent.com/dfordivam/703232aafb3ad3348095a0890cfe7911/raw/589014937e38d6d80f31c15d24b18b33a287d735/gistfile1.txt"
