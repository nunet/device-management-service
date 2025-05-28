package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: to be removed along with the old bootstrap nodes: #1089
func TestAddNewNodes(t *testing.T) {
	tests := []struct {
		name           string
		originalNodes  []string
		newNodes       []string
		expectedResult []string
	}{
		{
			name: "New nodes partially contained in original nodes",
			originalNodes: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWJMtMN1mTNRfgMqUygT7eSXamVzc9ihpSjeairm9PebmB",
			},
			expectedResult: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWJMtMN1mTNRfgMqUygT7eSXamVzc9ihpSjeairm9PebmB",
			},
		},
		{
			name: "Original nodes contain all of old nodes",
			originalNodes: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ",
			},
			expectedResult: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWJMtMN1mTNRfgMqUygT7eSXamVzc9ihpSjeairm9PebmB",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWKjSodxxi7UfRHzuk7eGgUF49MoPUCJvtva9K12TqDDsi",
			},
		},
		{
			name: "Original nodes contain only user's custom nodes",
			originalNodes: []string{
				"/user/custom/node/1", "/user/custom/node/2",
			},
			expectedResult: []string{
				"/user/custom/node/1", "/user/custom/node/2",
			},
		},
		{
			name: "Original nodes partially contain old nodes and custom user nodes",
			originalNodes: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/user/custom/node/1",
			},
			expectedResult: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/user/custom/node/1",
			},
		},
		{
			name: "Original nodes partially contain old nodes and no other",
			originalNodes: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
			},
			expectedResult: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWJMtMN1mTNRfgMqUygT7eSXamVzc9ihpSjeairm9PebmB",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWKjSodxxi7UfRHzuk7eGgUF49MoPUCJvtva9K12TqDDsi",
			},
		},
		{
			name: "Original nodes partially contain old nodes and new nodes",
			originalNodes: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
			},
			expectedResult: []string{
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
				"/dnsaddr/bootstrap.p2p.nunet.io/p2p/12D3KooWHzew9HTYzywFuvTHGK5Yzoz7qAhMfxagtCvhvjheoBQ3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := getBootstrapNodes(tt.originalNodes)
			assert.Equal(t, len(tt.expectedResult), len(result), "Expected length of result to match")
			assert.ElementsMatch(t, tt.expectedResult, result, "Expected nodes to match")
		})
	}
}
