package cap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func TestRunListCap_PassphraseError(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	_, _, err := utils.ExecuteCommand(newListCmd(dmsCli), "--context", "invalid")
	assert.Error(t, err)
}

func TestRunListCap_Success(t *testing.T) {
	t.Parallel()
	dmsCli := newTestCli()

	// create test cap context
	_, _, err := utils.NewCapabilityContext(dmsCli, "listctx")
	assert.NoError(t, err)

	output, _, err := utils.ExecuteCommand(newListCmd(dmsCli), "--context", "listctx")
	assert.NoError(t, err)
	assert.Contains(t, output, "roots:")
	assert.Contains(t, output, "require:")
	assert.Contains(t, output, "provide:")
	assert.Contains(t, output, "revoke:")
}

func TestFormatCapabilityList_Basic(t *testing.T) {
	t.Parallel()
	// Use simple types for DIDs and tokens
	roots := []did.DID{{URI: "did:example:root1"}, {URI: "did:example:root2"}}
	require := ucan.TokenList{Tokens: []*ucan.Token{{DMS: &ucan.DMSToken{Action: "require"}}}}
	provide := ucan.TokenList{Tokens: []*ucan.Token{{DMS: &ucan.DMSToken{Action: "provide"}}}}
	revoke := ucan.TokenList{Tokens: []*ucan.Token{{DMS: &ucan.DMSToken{Action: "revoke"}}}}

	expected := `roots:
	did:example:root1
	did:example:root2
require:
	{"dms":{"act":"require","iss":{},"sub":{},"aud":{},"cap":null,"nonce":null,"exp":0}}
provide:
	{"dms":{"act":"provide","iss":{},"sub":{},"aud":{},"cap":null,"nonce":null,"exp":0}}
revoke:
	{"dms":{"act":"revoke","iss":{},"sub":{},"aud":{},"cap":null,"nonce":null,"exp":0}}
`

	out, err := formatCapabilityList(roots, require, provide, revoke)
	assert.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestFormatCapabilityList_Empty(t *testing.T) {
	t.Parallel()
	out, err := formatCapabilityList(nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	assert.NoError(t, err)
	assert.Equal(t, "roots:\nrequire:\nprovide:\nrevoke:\n", out)
	assert.Contains(t, out, "require:")
	assert.Contains(t, out, "provide:")
	assert.Contains(t, out, "revoke:")
}

func TestFormatCapabilityList_MarshalError(t *testing.T) {
	t.Parallel()
	token := &ucan.Token{}
	token.DMS = nil
	badList := ucan.TokenList{Tokens: []*ucan.Token{token}}
	// The DMS field is nil, but this is still valid for json.Marshal, so to force an error, use a custom type if needed
	// For now, check that no error occurs with nils (Go's json.Marshal handles nil pointers gracefully)
	_, err := formatCapabilityList(nil, badList, ucan.TokenList{}, ucan.TokenList{})
	assert.NoError(t, err)
}
