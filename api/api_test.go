package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	//	"github.com/stretchr/testify/mock"
	//	"gitlab.com/nunet/device-management-service/integrations/oracle"

	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils"
)

var (
	debug       bool
	defaultPeer string
	mockHostID  = "Qm01testabcdefghjiklgfoobar123"
	mockAddr    = "/ip4/127.0.0.1/tcp/8080, /ip6/::1/udp/3000, /dns4/example.com/tcp/443/https"
	// DumpKademliaDHTHandler
	dumpKadDHTPeers int
	// ListDHTPeersHandler
	dhtPeers int
	// ListKadDHTPeersHandler
	kadDHTPeers      int
	mockInboundChats int
)

type MockHandler struct {
	buf *bytes.Buffer
}

func newMockHandler() *MockHandler {
	return &MockHandler{
		buf: new(bytes.Buffer),
	}
}

// type MockOracle struct {
// 	mock.Mock
// }
//
// func (o *MockOracle) WithdrawTokenRequest(req *oracle.RewardRequest) (*oracle.RewardResponse, error) {
// 	args := o.Called(req)
// 	return args.Get(0).(*oracle.RewardResponse), args.Error(1)
// }

func SetupMockRouter() *gin.Engine {
	m := newMockHandler()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	onboard := v1.Group("/onboarding")
	{
		onboard.GET("/metadata", m.GetMetadataHandler)
		onboard.GET("/status", m.OnboardStatusHandler)
		onboard.GET("/provisioned", m.ProvisionedCapacityHandler)
		onboard.GET("/address/new", m.CreatePaymentAddressHandler)
		onboard.POST("/onboard", OnboardHandler)
		onboard.POST("/resource-config", m.ResourceConfigHandler)
		onboard.DELETE("/offboard", m.OffboardHandler)
	}
	device := v1.Group("/device")
	{
		device.GET("/status", m.DeviceStatusHandler)
		device.POST("/status", m.ChangeDeviceStatusHandler)
	}
	vm := v1.Group("/vm")
	{
		vm.POST("/start-default", m.StartDefaultHandler)
		vm.POST("/start-custom", m.StartCustomHandler)
	}
	run := v1.Group("/run")
	{
		run.GET("/checkpoints", m.ListCheckpointHandler)
		run.GET("/deploy", m.DeploymentRequestHandler)
		run.POST("/request-service", m.RequestServiceHandler)
	}
	tx := v1.Group("/transactions")
	{
		tx.GET("", m.GetJobTxHashesHandler)
		tx.POST("/request-reward", m.RequestRewardHandler)
		tx.POST("/send-status", m.SendTxStatusHandler)
		tx.POST("/update-status", m.UpdateTxStatusHandler)
	}
	tele := v1.Group("/telemetry")
	{
		tele.GET("/free", m.GetFreeResourcesHandler)
	}
	if debug == true {
		dht := v1.Group("/dht")
		{
			// dht.GET("", m.DumpDHTHandler)
			dht.GET("/update", m.ManualDHTUpdateHandler)
		}
		kadDHT := v1.Group("/kad-dht")
		{
			kadDHT.GET("", m.DumpKademliaDHTHandler)
		}
		v1.GET("/ping", m.PingPeerHandler)
		v1.GET("/oldping", m.OldPingPeerHandler)
		v1.GET("/cleanup", m.CleanupPeerHandler)
	}
	p2p := v1.Group("/peers")
	{
		p2p.GET("", m.ListPeersHandler)
		p2p.GET("/dht", m.ListDHTPeersHandler)
		p2p.GET("/self", m.SelfPeerInfoHandler)
	}
	return router
}

func WriteMockMetadata(fs afero.Fs) (string, error) {
	path := utils.GetMetadataFilePath()
	metadata := &models.Metadata{
		Name:            "ExampleName",
		UpdateTimestamp: 1625072042,
		Network:         "192.168.1.1",
		PublicKey:       "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3",
		NodeID:          "node-123",
		AllowCardano:    true,
		Dashboard:       "http://localhost:3000",
	}
	content, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("could not marshal data into mock metadata: %w", err)
	}
	err = afero.WriteFile(fs, path, content, 0644)
	if err != nil {
		return "", fmt.Errorf("could not write content to mock metadata: %w", err)
	}
	buf := bytes.NewBuffer(content)
	return buf.String(), nil
}

func startMockWebSocketServer() *http.Server {
	upgrader := websocket.Upgrader{}
	server := &http.Server{
		Addr: "localhost:8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			for {
				messageType, p, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if err := conn.WriteMessage(messageType, p); err != nil {
					return
				}
			}
		}),
	}

	go server.ListenAndServe()
	return server
}

func TestCardanoAddressRoute(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/onboarding/address/new", nil)
	router.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, string(body), "address")
	assert.Contains(t, string(body), "mnemonic")

	var jsonMap map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &jsonMap)

	assert.NotEmpty(t, jsonMap)
}

func TestEthereumAddressRoute(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/onboarding/address/new?blockchain=ethereum", nil)
	router.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, string(body), "address")
	assert.Contains(t, string(body), "private_key")

	var jsonMap map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &jsonMap)

	assert.NotEmpty(t, jsonMap)
}

func TestProvisionedRoute(t *testing.T) {
	router := SetupMockRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/onboarding/provisioned", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "cpu")
	assert.Contains(t, w.Body.String(), "memory")
}

func TestOnboardEmptyRequest(t *testing.T) {
	expectedResponse := `{"error":"invalid request data"}`

	rServer := NewRestServer(nil, nil, nil, 0)
	onboarding := rServer.router.Group("/api/v1/onboarding")
	onboarding.POST("/onboard", OnboardHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/onboarding/onboard", nil)
	rServer.router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
	assert.Equal(t, expectedResponse, w.Body.String())
}

func TestOnboard_CapacityTooLowTooHigh(t *testing.T) {
	onboarding.FS = afero.NewMemMapFs()
	onboarding.AFS = &afero.Afero{Fs: onboarding.FS}
	onboarding.AFS.Mkdir("/etc/nunet", 0777)
	expectedCPUResponse := "CPU should be between 10% and 90% of the available CPU"
	expectedRAMResponse := "memory should be between 10% and 90% of the available memory"

	router := SetupMockRouter()
	w := httptest.NewRecorder()

	type TestRequestPayload struct {
		Memory      int64  `json:"memory"`
		CPU         int64  `json:"cpu"`
		Channel     string `json:"channel"`
		PaymentAddr string `json:"payment_addr"`
		Cardano     bool   `json:"cardano"`
	}
	totalCpu := resources.GetTotalProvisioned().CPU
	totalMem := resources.GetTotalProvisioned().Memory

	// test too low CPU (less than 10% of machine resource)
	lowCPUTestPayload := TestRequestPayload{
		Memory:      int64(float64(totalMem) * 0.5),  // 50% acceptable
		CPU:         int64(float64(totalCpu) * 0.05), // 5% unacceptable
		Channel:     "nunet-test",
		PaymentAddr: "addr1q99z75su8d8w0jv6drfnr3tuyycflcg4pqpvpnvfzlmmdl7m4nxzjpxhvx477ruhswnrkuqju0kyhx4mvwr0geqyfass7rwta8",
		Cardano:     false,
	}
	jsonPayload, _ := json.Marshal(lowCPUTestPayload)
	req, _ := http.NewRequest("POST", "/api/v1/onboarding/onboard", bytes.NewBuffer((jsonPayload)))
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), expectedCPUResponse)

	// test too high CPU (more than 90% of machine resource)
	highCPUTestPayload := TestRequestPayload{
		Memory:      int64(float64(totalMem) * 0.5),  // 50% acceptable
		CPU:         int64(float64(totalCpu) * 0.95), // 95% unacceptable
		Channel:     "nunet-test",
		PaymentAddr: "addr1q99z75su8d8w0jv6drfnr3tuyycflcg4pqpvpnvfzlmmdl7m4nxzjpxhvx477ruhswnrkuqju0kyhx4mvwr0geqyfass7rwta8",
		Cardano:     false,
	}
	jsonPayload, _ = json.Marshal(highCPUTestPayload)
	req, _ = http.NewRequest("POST", "/api/v1/onboarding/onboard", bytes.NewBuffer((jsonPayload)))
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), expectedCPUResponse)

	// test too low memory (less than 10% of machine resource)
	lowRAMTestPayload := TestRequestPayload{
		Memory:      int64(float64(totalMem) * 0.05), // 5% unacceptable
		CPU:         int64(float64(totalCpu) * 0.5),  // 50% acceptable
		Channel:     "nunet-test",
		PaymentAddr: "addr1q99z75su8d8w0jv6drfnr3tuyycflcg4pqpvpnvfzlmmdl7m4nxzjpxhvx477ruhswnrkuqju0kyhx4mvwr0geqyfass7rwta8",
		Cardano:     false,
	}
	jsonPayload, _ = json.Marshal(lowRAMTestPayload)
	req, _ = http.NewRequest("POST", "/api/v1/onboarding/onboard", bytes.NewBuffer((jsonPayload)))
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), expectedRAMResponse)

	// test too high VPU (more than 90% of machine resource)
	highRAMTestPayload := TestRequestPayload{
		Memory:      int64(float64(totalMem) * 0.95), // 95% unacceptable
		CPU:         int64(float64(totalCpu) * 0.5),  // 50% acceptable
		Channel:     "nunet-test",
		PaymentAddr: "addr1q99z75su8d8w0jv6drfnr3tuyycflcg4pqpvpnvfzlmmdl7m4nxzjpxhvx477ruhswnrkuqju0kyhx4mvwr0geqyfass7rwta8",
		Cardano:     false,
	}
	jsonPayload, _ = json.Marshal(highRAMTestPayload)
	req, _ = http.NewRequest("POST", "/api/v1/onboarding/onboard", bytes.NewBuffer((jsonPayload)))
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), expectedRAMResponse)

}
