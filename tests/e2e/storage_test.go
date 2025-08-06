//go:build storagetst || !unit

package itest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/dms"
)

func TLSGlusterGenerator(t *testing.T) {
	os.Setenv("GOLOG_LOG_LEVEL", "debug")
	password := "password"
	here := getCurrentFileDirectory()

	rootDir := filepath.Join("testdata", "storage", "dms1")
	rootDir2 := filepath.Join("testdata", "storage", "dms2")
	rootDir3 := filepath.Join("testdata", "storage", "dms3")

	_ = os.RemoveAll(filepath.Join(here, rootDir))
	_ = os.RemoveAll(filepath.Join(here, rootDir2))
	_ = os.RemoveAll(filepath.Join(here, rootDir3))

	_ = os.RemoveAll("/tmp/client_gluster_tls_certs")

	node1Config := createConfig(rootDir, 9992, []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 9087), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 9087)}, []string{})
	cli1 := newClient(t, node1Config)
	cli1.newKey(t, "dms", password)
	cli1.newCap(t, "dms", password)
	dms1DID := cli1.getDID(t, fmt.Sprintf("%s.cap", "dms"), password)
	require.NotEmpty(t, dms1DID)

	node3Config := createConfig(rootDir3, 9997, []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 9095), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 9095)}, []string{})
	cli3 := newClient(t, node3Config)
	cli3.newKey(t, "dms", password)
	cli3.newCap(t, "dms", password)
	dms3DID := cli3.getDID(t, fmt.Sprintf("%s.cap", "dms"), password)
	require.NotEmpty(t, dms3DID)

	node2Config := createConfig(rootDir2, 9994, []string{fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 9089), fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", 9089)}, []string{})
	cli2 := newClient(t, node2Config)
	cli2.newKey(t, "dms", password)
	cli2.newCap(t, "dms", password)
	dms2DID := cli2.getDID(t, fmt.Sprintf("%s.cap", "dms"), password)
	require.NotEmpty(t, dms2DID)

	// anchor them as root together
	cli1.addRootAnchor(t, "dms", dms2DID, password)
	cli1.addRootAnchor(t, "dms", dms3DID, password)

	cli2.addRootAnchor(t, "dms", dms1DID, password)
	cli2.addRootAnchor(t, "dms", dms3DID, password)

	cli3.addRootAnchor(t, "dms", dms1DID, password)
	cli3.addRootAnchor(t, "dms", dms2DID, password)

	const home = "/home/"

	// dont move it above
	node2Config.General.StorageMode = true
	node2Config.General.UserDir = home
	node2Config.General.DataDir = home
	node2Config.General.WorkDir = home
	hostname, _ := os.Hostname()
	node2Config.General.StorageCADirectory = "/home/ca_dir"
	node2Config.General.StorageBricksDir = "/home/bricks_dir"
	node2Config.General.StorageGlusterfsHostname = hostname
	node2Config.Observability.LogFile = "/home/logs.txt"
	node2Config.Profiler.Port = 6066

	jsonData2, err := json.MarshalIndent(node2Config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(here, rootDir2, "dms_config.json"), jsonData2, 0o644)
	require.NoError(t, err)

	binaryPath := binaryName

	tlsServerContainerName := "glusterfs-server-tls-test"
	err = runGlusterContainer(tlsServerContainerName)
	require.NoError(t, err)
	what := filepath.Join(here, binaryPath)
	fmt.Println("what ", what)
	err = copyToContainer(tlsServerContainerName, what, "/usr/bin")
	require.NoError(t, err)

	err = copyToContainer(tlsServerContainerName, filepath.Join(here, rootDir2, "cap"), "/home/")
	require.NoError(t, err)
	err = copyToContainer(tlsServerContainerName, filepath.Join(here, rootDir2, "key"), "/home/")
	require.NoError(t, err)
	err = copyToContainer(tlsServerContainerName, filepath.Join(here, rootDir2, "dms_config.json"), "/")
	require.NoError(t, err)

	commands := [][]string{
		{"mkdir", "/client_certs"},
		{"mkdir", "-p", "/home/ca_dir/glusterfs_nodes"},
		{"mkdir", "-p", "/home/bricks_dir"},
		{"chmod", "777", "/home/bricks_dir/"},
		{"touch", "/var/lib/glusterd/secure-access"},
	}
	err = runGlusterCommands(tlsServerContainerName, commands)
	require.NoError(t, err)

	err = generateCerts(hostname, "/tmp/gluster_tls_certs")
	require.NoError(t, err)

	err = copyToContainer(tlsServerContainerName, "/tmp/gluster_tls_certs/glusterfs.key", "/etc/pki/tls/")
	require.NoError(t, err)
	err = copyToContainer(tlsServerContainerName, "/tmp/gluster_tls_certs/glusterfs.pem", "/etc/pki/tls/")
	require.NoError(t, err)
	err = copyToContainer(tlsServerContainerName, "/tmp/gluster_tls_certs/glusterfs.pem", "/etc/pki/tls/glusterfs.ca")
	require.NoError(t, err)

	// copy also to dms ca_dir
	err = copyToContainer(tlsServerContainerName, "/tmp/gluster_tls_certs/glusterfs.pem", "/home/ca_dir/glusterfs_nodes/")
	require.NoError(t, err)

	// run first dms
	dms1, err := dms.NewDMS(cli1.fs, node1Config, cli1.env, password, "dms")
	require.NoError(t, err)
	assert.NotNil(t, dms1)
	err = dms1.Run()
	require.NoError(t, err)
	time.Sleep(7 * time.Second)

	// get dms1 multiaddr and construct a bootstrap so dms2 can connect to
	multiAddr, err := dms1.P2P.GetMultiaddr()
	require.NoError(t, err)

	bootstrap := []string{}
	for _, v := range multiAddr {
		bootstrap = append(bootstrap, v.String())
	}

	// run third dms
	node3Config.BootstrapPeers = bootstrap
	dms3, err := dms.NewDMS(cli3.fs, node3Config, cli3.env, password, "dms")
	require.NoError(t, err)
	assert.NotNil(t, dms3)
	err = dms3.Run()
	require.NoError(t, err)
	time.Sleep(7 * time.Second)

	// run dms in container
	err = runBinaryInContainer(tlsServerContainerName, "dms", []string{"run", "--context", "dms"}, []string{"DMS_PASSPHRASE=password", "BOOTSTRAP_PEERS=" + strings.Join(bootstrap, ",")}, "/home/dms_log.txt")
	require.NoError(t, err)
	time.Sleep(10 * time.Second)

	// broadcast
	result := cli1.broadcast(t, "dms", password)
	require.Equal(t, 3, countDIDOccurrences(result))

	// onboard
	cli1.onboard(t, "dms", password)
	cli3.onboard(t, "dms", password)

	// generate client certs
	err = generateCerts("clientXyx", "/tmp/client_gluster_tls_certs")
	require.NoError(t, err)

	_, err = cli1.createVolume(t, "volumenunet123", "/tmp/client_gluster_tls_certs/glusterfs.pem", "/tmp/client_gluster_tls_certs/", "dms", password, dms2DID)
	require.NoError(t, err)

	_, err = cli1.startVolume(t, "volumenunet123", "dms", password, dms2DID)
	require.NoError(t, err)

	err = copyToContainer(tlsServerContainerName, "/tmp/client_gluster_tls_certs/", "/client_certs/")
	require.NoError(t, err)

	commands2 := [][]string{
		{"gluster", "volume", "set", "volumenunet123", "diagnostics.client-log-level", "DEBUG"},
		{"cp", "/home/ca_dir/glusterfs.ca", "/etc/pki/tls/glusterfs.ca"},
		{"systemctl", "restart", "glusterd"},
	}
	err = runGlusterCommands(tlsServerContainerName, commands2)
	require.NoError(t, err)

	ensemble := filepath.Join(here, rootDir2, "../../", "ensembles", "firefly.yaml")
	err = copyToContainer(tlsServerContainerName, ensemble, "/home/")
	require.NoError(t, err)

	err = runBinaryInContainer(tlsServerContainerName, "dms", []string{"actor", "cmd", "--context", "dms", "/dms/node/deployment/new", "--spec-file", "/home/firefly.yaml", "--timeout", "2m"}, []string{"DMS_PASSPHRASE=password"}, "/home/run_ensemble.txt")
	require.NoError(t, err)

	time.Sleep(2 * time.Minute)
	_ = deleteGlusterContainer(tlsServerContainerName)
}

func generateCerts(hostname string, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	keyPath := fmt.Sprintf("%s/glusterfs.key", dir)
	certPath := fmt.Sprintf("%s/glusterfs.pem", dir)

	genKeyCmd := exec.Command("openssl", "genrsa", "-out", keyPath, "2048")
	genKeyCmd.Stderr = os.Stderr
	if err := genKeyCmd.Run(); err != nil {
		return fmt.Errorf("error generating key: %v", err)
	}

	subj := fmt.Sprintf("/CN=%s", hostname)
	genCertCmd := exec.Command("openssl", "req", "-new", "-x509", "-key", keyPath, "-subj", subj, "-out", certPath)
	genCertCmd.Stderr = os.Stderr
	if err := genCertCmd.Run(); err != nil {
		return fmt.Errorf("error generating certificate: %v", err)
	}

	return nil
}
