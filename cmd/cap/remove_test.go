package cap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
)

func TestRemoveCmd_InvalidArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no context",
			args: []string{},
		},
		{
			name: "no anchor type",
			args: []string{"--context", userCtx},
		},
		{
			name: "invalid did",
			args: []string{"--context", userCtx, "--root", "not-a-did"},
		},
		{
			name: "invalid provide token format",
			args: []string{"--context", userCtx, "--provide", "not-valid-json"},
		},
		{
			name: "invalid require token format",
			args: []string{"--context", userCtx, "--require", "not-valid-json"},
		},
	}

	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newRemoveCmd(dmsCli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestRemoveCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newRemoveCmd(dmsCli),
		"--context", "invalid",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.Error(t, err)
}

func TestRemoveCmd_Success(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, "removectx")
	assert.NoError(t, err)

	// anchor
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", "removectx",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)

	// remove
	_, _, err = utils.ExecuteCommand(newRemoveCmd(dmsCli),
		"--context", "removectx",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)
}
