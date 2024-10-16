package libp2p

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// ResolveDNS resolves a DNS query using the provided resolver
func resolveDNS(query *dns.Msg, records map[string]string) *dns.Msg {
	// Create a response message
	m := new(dns.Msg)
	m.SetReply(query)

	for _, question := range query.Question {
		if question.Qtype != dns.TypeA {
			// We only support A records
			m.SetRcode(query, dns.RcodeNotImplemented)
			continue
		}

		ip, ok := records[question.Name]
		if !ok {
			// Not found in our map, set answer to NXDOMAIN
			m.SetRcode(query, dns.RcodeNameError)
			continue
		}

		// Found record, add A record to the answer section
		a := &dns.A{
			Hdr: dns.RR_Header{Name: question.Name, Rrtype: dns.TypeA, Class: dns.ClassINET},
			A:   net.ParseIP(ip),
		}

		m.Answer = append(m.Answer, a)
	}

	return m
}

// HandleDNSQuery handles a DNS query by parsing the UDP packet, resolving the query, and sending a response
func handleDNSQuery(packet []byte, records map[string]string) ([]byte, error) {
	// Parse the UDP packet into a DNS message
	msg := new(dns.Msg)
	err := msg.Unpack(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to decode DNS message: %w", err)
	}

	// Resolve the DNS query
	response := resolveDNS(msg, records)
	log.Debug("DNS query resolved successfully", "response", response)

	// Encode the response message into a UDP packet
	responseBytes, err := response.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to encode DNS response: %w", err)
	}

	return responseBytes, nil
}
