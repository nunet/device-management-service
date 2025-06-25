package cap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestGrantCmd_InvalidArgs(t *testing.T) {
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

	cli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(cli, userCtx)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := utils.ExecuteCommand(newGrantCmd(cli), tt.args...)
			assert.Error(t, err)
		})
	}
}

func TestGrantCmd_PassphraseError(t *testing.T) {
	t.Parallel()
	cli := newTestCli()

	_, _, err := utils.ExecuteCommand(newGrantCmd(cli),
		"--context", "invalid",
		"--cap", "/public",
		"--duration", "1h",
		"did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.Error(t, err)
}

func TestGrantCmd_Success(t *testing.T) {
	t.Parallel()
	cli := newTestCli()

	// create test cap context
	_, userDID, _ := utils.NewCapabilityContext(cli, userCtx)

	tokenStr, _, err := utils.ExecuteCommand(newGrantCmd(cli),
		"--context", userCtx,
		"--cap", "/public",
		"--topic", "testtopic",
		"--duration", "1h",
		"--depth", "1",
		"did:key:z6MkwQpRz8b7vJY4N1k2",
	)
	assert.NoError(t, err)

	// verify token
	var grantToken ucan.TokenList
	assert.NoError(t, json.Unmarshal([]byte(tokenStr), &grantToken))

	assert.Equal(t, ucan.Delegate, grantToken.Tokens[0].Action())
	assert.Equal(t, userDID, grantToken.Tokens[0].Issuer())
	assert.Equal(t, "did:key:z6MkwQpRz8b7vJY4N1k2", grantToken.Tokens[0].Subject().String())
	assert.Contains(t, grantToken.Tokens[0].Capability(), ucan.Capability("/public"))
	assert.Contains(t, grantToken.Tokens[0].Topic(), ucan.Capability("testtopic"))
}
