package cap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestRevokeCmd_MissingArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no token",
			args: []string{"--context", userCtx},
		},
		{
			name: "no context",
			args: []string{"{\"token\": \"some-token\"}"},
		},
	}

	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newRevokeCmd(dmsCli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestRevokeCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newRevokeCmd(dmsCli),
		"--context", "invalid",
		"{}",
	)
	assert.Error(t, err)
}

func TestRevokeCmd_Success(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	// grant
	grantTokensStr, _, err := utils.ExecuteCommand(newGrantCmd(dmsCli),
		"--context", userCtx,
		"--cap", "/public",
		"--duration", "1h",
		"--depth", "1",
		"did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)

	// revoke
	revokeTokensStr, _, err := utils.ExecuteCommand(newRevokeCmd(dmsCli),
		"--context", userCtx,
		grantTokensStr,
	)
	assert.NoError(t, err)

	// Check if the token was revoked
	var revokeTokens ucan.Token
	assert.NoError(t, json.Unmarshal([]byte(revokeTokensStr), &revokeTokens))

	var grantTokens ucan.TokenList
	assert.NoError(t, json.Unmarshal([]byte(grantTokensStr), &grantTokens))

	assert.Equal(t, ucan.Revoke, revokeTokens.Action())
	assert.Equal(t, grantTokens.Tokens[0].Issuer(), revokeTokens.Issuer())
	assert.Equal(t, grantTokens.Tokens[0].Subject(), revokeTokens.Subject())
}
