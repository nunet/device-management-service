//go:build acceptance || !unit

package acceptance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/tests/acceptance/config"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
	dutils "gitlab.com/nunet/device-management-service/utils"
)

func TestPrepareENV(t *testing.T) {
	cfg, err := config.Get()
	assert.NoError(t, err)

	clients, err := utils.ConnectToClients(cfg)
	assert.NoError(t, err)

	// the prefix name can come from a pipeline unique id or something
	// so we dont have conflict between 2 different pipelines
	nodes, err := utils.CreateNodes(clients, 2, "ubuntu/22.04", "vm-prefix-name")
	assert.NoError(t, err)

	here := dutils.CurrentFileDirectory()
	remoteDMSPath := "/usr/local/bin/dms"
	localPath := filepath.Join(here, "..", "builds", "dms_linux_amd64")

	for _, v := range nodes {
		err = v.UploadFile(localPath, remoteDMSPath, 0o755)
		assert.NoError(t, err)
		_, err = v.RunCMD([]string{"chmod", "+x", "/usr/local/bin/dms"})
		assert.NoError(t, err)
	}

	err = nodes[0].RunCMDBackground([]string{"sh", "-c", "DMS_PASSPHRASE=123 /usr/local/bin/dms run"})
	assert.NoError(t, err)

	for _, v := range nodes {
		err = v.Destroy()
		assert.NoError(t, err)
	}
}
