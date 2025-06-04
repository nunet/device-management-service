package cap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestAnchorCmd_InvalidArgs(t *testing.T) {
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
		{
			name: "invalid revoke token format",
			args: []string{"--context", userCtx, "--revoke", "not-valid-json"},
		},
	}

	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newAnchorCmd(dmsCli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestAnchorCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", "invalid",
		"--root", "did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.Error(t, err)
}

func TestAnchorCmd_Success(t *testing.T) {
	t.Parallel()

	dmsCli := newTestCli()

	// create test cap contexts
	_, orgDID, _ := utils.NewCapabilityContext(dmsCli, orgCtx)
	_, userDID, _ := utils.NewCapabilityContext(dmsCli, userCtx)
	_, _, _ = utils.NewCapabilityContext(dmsCli, dmsCtx)
	_, _, err := utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", dmsCtx,
		"--root", userDID.String(),
	)
	assert.NoError(t, err)

	// orgDID grants capablities to userDID
	token1, _, err := utils.ExecuteCommand(newGrantCmd(dmsCli),
		"--context", orgCtx,
		"--cap", "/public",
		"--duration", "1h",
		"--depth", "1",
		userDID.String(),
	)
	assert.NoError(t, err)

	// userDID provides granted token
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", userCtx,
		"--provide", token1,
	)
	assert.NoError(t, err)

	// userDID grants capablities to orgDID
	token2, _, err := utils.ExecuteCommand(newGrantCmd(dmsCli),
		"--context", userCtx,
		"--cap", "/public",
		"--duration", "1h",
		"--depth", "1",
		orgDID.String(),
	)
	assert.NoError(t, err)

	// dmsDID requires granted token
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", dmsCtx,
		"--require", token2,
	)
	assert.NoError(t, err)

	// userDID creates revoke token
	token3, _, err := utils.ExecuteCommand(newRevokeCmd(dmsCli),
		"--context", userCtx,
		token2,
	)
	assert.NoError(t, err)

	// userDID revokes revoke token
	_, _, err = utils.ExecuteCommand(newAnchorCmd(dmsCli),
		"--context", userCtx,
		"--revoke", token3,
	)
	assert.NoError(t, err)

	var reqToken ucan.TokenList
	var provideToken ucan.TokenList
	var revokeToken ucan.Token

	assert.NoError(t, json.Unmarshal([]byte(token1), &provideToken))
	assert.NoError(t, json.Unmarshal([]byte(token2), &reqToken))
	assert.NoError(t, json.Unmarshal([]byte(token3), &revokeToken))

	// verify anchors
	userCapCtx, err := utils.LoadCapabilityContext(dmsCli, userCtx)
	assert.NoError(t, err)
	userRoots, userRequireTokens, userProvideTokens, userRevokeTokens := userCapCtx.ListRoots()
	dmsCapCtx, err := utils.LoadCapabilityContext(dmsCli, dmsCtx)
	assert.NoError(t, err)
	dmsRoots, dmsRequireTokens, dmsProvideTokens, dmsRevokeTokens := dmsCapCtx.ListRoots()
	assert.Empty(t, userRoots)
	assert.Empty(t, userRequireTokens.Tokens)
	assert.Equal(t, provideToken, userProvideTokens)
	assert.Equal(t, ucan.TokenList{Tokens: []*ucan.Token{&revokeToken}}, userRevokeTokens)
	assert.Equal(t, []did.DID{userDID}, dmsRoots)
	assert.Equal(t, reqToken, dmsRequireTokens)
	assert.Empty(t, dmsProvideTokens.Tokens)
	assert.Empty(t, dmsRevokeTokens.Tokens)
}
