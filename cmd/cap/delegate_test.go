package cap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestDelegateCmd_InvalidArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no args",
			args: []string{},
		},
		{
			name: "no context",
			args: []string{"did:key:z6MkwQpRz8b7vJY4N1k2"},
		},
		{
			name: "no expiration",
			args: []string{
				"--context", userCtx,
				"did:key:z6MkwQpRz8b7vJY4N1k2",
			},
		},
		{
			name: "invalid did",
			args: []string{
				"--context", userCtx,
				"--duration", "1h",
				"not-a-did",
			},
		},
	}

	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newDelegateCmd(dmsCli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestDelegateCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newDelegateCmd(dmsCli),
		"--context", "invalid",
		"--cap", "/public",
		"--duration", "1h",
		"did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.Error(t, err)
}

func TestDelegateCmd_Success(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	// create capabilities
	_, userDID, _ := utils.NewCapabilityContext(dmsCli, userCtx)
	_, dmsDID, _ := utils.NewCapabilityContext(dmsCli, dmsCtx)
	_, _, _ = utils.NewCapabilityContext(dmsCli, orgCtx)

	// org grants capabilities to user
	token, _, err := utils.ExecuteCommand(newGrantCmd(dmsCli),
		"--context", orgCtx,
		"--cap", "/public",
		"--cap", "/broadcast",
		"--topic", "testtopic",
		"--depth", "2",
		"--duration", "2h",
		userDID.String(),
	)
	assert.NoError(t, err)

	// user provides token
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", userCtx,
		"--provide", token,
	)
	assert.NoError(t, err)

	// user delegates capabilities to dms
	delegateTokenStr, _, err := utils.ExecuteCommand(newDelegateCmd(dmsCli),
		"--context", userCtx,
		"--cap", "/public",
		"--cap", "/broadcast",
		"--topic", "testtopic",
		"--duration", "1h",
		dmsDID.String(),
	)
	assert.NoError(t, err)

	// verify token
	var delegateToken ucan.TokenList
	assert.NoError(t, json.Unmarshal([]byte(delegateTokenStr), &delegateToken))

	assert.Equal(t, ucan.Delegate, delegateToken.Tokens[0].Action())
	assert.Equal(t, userDID, delegateToken.Tokens[0].Issuer())
	assert.Equal(t, dmsDID, delegateToken.Tokens[0].Subject())
	assert.Contains(t, delegateToken.Tokens[0].Capability(), ucan.Capability("/public"))
	assert.Contains(t, delegateToken.Tokens[0].Capability(), ucan.Capability("/broadcast"))
	assert.Contains(t, delegateToken.Tokens[0].Topic(), ucan.Capability("testtopic"))
}
