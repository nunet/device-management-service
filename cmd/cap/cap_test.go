package cap

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func TestMain(m *testing.M) {
	// Use fast scrypt params for tests
	keystore.SetTestScryptParams(1<<10, 1, 1) // N=1024, R=1, P=1 (significantly faster)

	code := m.Run()

	// Reset scrypt params to defaults after tests
	keystore.ResetScryptParamsToDefaults()

	os.Exit(code)
}

const (
	orgCtx  = "orgctx"
	userCtx = "userctx"
	dmsCtx  = "dmsctx"
)

func newTestCli(opts ...func(*cli.DmsCLI)) *cli.DmsCLI {
	defaults := []func(*cli.DmsCLI){}

	env := env.NewMockEnvironment()
	err := env.Setenv("DMS_PASSPHRASE", "pass")
	if err == nil {
		defaults = append(defaults, cli.WithEnv(env))
	}

	fs := afero.NewMemMapFs()
	cfg := &config.Config{General: config.General{
		UserDir: "/tmp/nunet/user",
		WorkDir: "/tmp/nunet/work",
		DataDir: "/tmp/nunet/data",
	}}

	defaults = append(defaults, cli.WithFS(fs), cli.WithConfig(cfg))

	dmsCli := cli.New(append(defaults, opts...)...)

	return dmsCli
}

func TestNewCapCmd(t *testing.T) {
	t.Parallel()

	dmsCli := newTestCli()
	cfg, err := dmsCli.Config()
	assert.NoError(t, err)

	afs := afero.Afero{Fs: dmsCli.FS()}

	cmd := NewCapCmd(afs, dmsCli.Env(), cfg)
	_, _, err = utils.ExecuteCommand(cmd)
	assert.NoError(t, err)
}
