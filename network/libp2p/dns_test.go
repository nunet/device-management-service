// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDomainAlice = "alice.com"
	testDomainBob   = "bob.com"
	testAliceIP     = "192.168.1.1"
	testBobIP       = "192.168.1.2"
)

func setupTestRecords(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		testDomainAlice: testAliceIP,
		testDomainBob:   testBobIP,
	}
}

func TestResolveDNS(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(testDomainAlice), dns.TypeA)

	response := resolveDNS(query, records)

	assert.Equal(t, 1, len(response.Answer))
	assert.Equal(t, dns.TypeA, response.Answer[0].Header().Rrtype)
	aRecord, ok := response.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.Equal(t, testAliceIP, aRecord.A.String())
}

func TestDNS_NonExistentDomain(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn("nonexistent.com"), dns.TypeA)

	response := resolveDNS(query, records)

	assert.Equal(t, 0, len(response.Answer))
	assert.Equal(t, dns.RcodeServerFailure, response.Rcode)
}

func TestDNS_NonARecord(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(testDomainAlice), dns.TypeMX)

	response := resolveDNS(query, records)

	assert.Equal(t, 0, len(response.Answer))
	assert.Equal(t, dns.RcodeNotImplemented, response.Rcode)
}

func TestDNS_MultipleQuestions(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	query := new(dns.Msg)
	query.Question = []dns.Question{
		{
			Name:   dns.Fqdn(testDomainAlice),
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		},
		{
			Name:   dns.Fqdn(testDomainBob),
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		},
	}

	response := resolveDNS(query, records)

	assert.Equal(t, 2, len(response.Answer))

	aRecord1, ok := response.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.Equal(t, dns.Fqdn(testDomainAlice), aRecord1.Header().Name)
	assert.Equal(t, testAliceIP, aRecord1.A.String())

	aRecord2, ok := response.Answer[1].(*dns.A)
	assert.True(t, ok)
	assert.Equal(t, dns.Fqdn(testDomainBob), aRecord2.Header().Name)
	assert.Equal(t, testBobIP, aRecord2.A.String())
}

func TestDNS_ValidQuery(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(testDomainAlice), dns.TypeA)

	// Convert query to wire format
	packet, err := query.Pack()
	require.NoError(t, err)

	responseBytes, err := handleDNSQuery(packet, records)
	assert.NoError(t, err)
	assert.NotNil(t, responseBytes)

	// Parse response and check
	response := new(dns.Msg)
	err = response.Unpack(responseBytes)
	require.NoError(t, err)

	assert.Equal(t, 1, len(response.Answer))
	aRecord, ok := response.Answer[0].(*dns.A)
	assert.True(t, ok)
	assert.Equal(t, testAliceIP, aRecord.A.String())
}

func TestDNS_InvalidQuery(t *testing.T) {
	t.Parallel()

	records := setupTestRecords(t)
	invalidPacket := []byte("invalid DNS packet")

	_, err := handleDNSQuery(invalidPacket, records)
	assert.Error(t, err)
}
