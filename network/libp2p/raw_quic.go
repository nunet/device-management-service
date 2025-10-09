// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"net"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/transport/quicreuse"
	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/quic-go/quic-go"
)

type RawQUICTransport struct {
	*quic.Transport
	listener *interceptingListener
	network  *Libp2p

	listenerReady chan struct{}
}

type interceptingListener struct {
	intercept []string

	acceptQueue chan *quic.Conn
	quicreuse.QUICListener
}

func NewRawQUICTransport(udpConn *net.UDPConn) *RawQUICTransport {
	return &RawQUICTransport{Transport: &quic.Transport{Conn: udpConn}, listenerReady: make(chan struct{})}
}

func (t *RawQUICTransport) Listen(tlsConf *tls.Config, conf *quic.Config) (quicreuse.QUICListener, error) {
	wrappedConf := &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			if slices.Contains(info.SupportedProtos, "raw") {
				priv, err := t.network.config.PrivateKey.Raw()
				if err != nil {
					return nil, fmt.Errorf("failed to get priv key to generate tls cert: %w", err)
				}

				var cert *tls.Certificate
				cert, err = generateSelfSignedCert(ed25519.PrivateKey(priv), []string{info.ServerName})
				if err != nil {
					return nil, fmt.Errorf("failed to generate self signed cert: %w", err)
				}
				return &tls.Config{
					ClientAuth:            tls.RequireAnyClientCert,
					Certificates:          []tls.Certificate{*cert},
					NextProtos:            []string{"raw"},
					InsecureSkipVerify:    false,
					VerifyPeerCertificate: makeVerifySubnetPeerCertificateFn(t.network),
				}, nil
			}
			// use libp2p's tls.Config
			if tlsConf.GetConfigForClient != nil {
				return tlsConf.GetConfigForClient(info)
			}
			return tlsConf, nil
		},
	}
	ln, err := t.Transport.Listen(wrappedConf, conf)
	if err != nil {
		return nil, err
	}
	t.listener = newInterceptingListener(ln, []string{"raw"})

	close(t.listenerReady)

	return t.listener, nil
}

func newInterceptingListener(ln quicreuse.QUICListener, intercept []string) *interceptingListener {
	return &interceptingListener{
		intercept:    intercept,
		acceptQueue:  make(chan *quic.Conn, 32),
		QUICListener: ln,
	}
}

func (l *interceptingListener) Accept(ctx context.Context) (*quic.Conn, error) {
start:
	conn, err := l.QUICListener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	if conn.ConnectionState().TLS.NegotiatedProtocol == "raw" {
		log.Debugf("intercepting a raw connection from: %s", conn.RemoteAddr())
		l.acceptQueue <- conn
		goto start
	}
	log.Debugf("accepting a non-raw connection from: %s", conn.RemoteAddr())
	return conn, nil
}

type quicAddrFilter struct{}

func (f *quicAddrFilter) filterQUICIPv4(_ peer.ID, maddrs []ma.Multiaddr) []ma.Multiaddr {
	return ma.FilterAddrs(maddrs, func(addr ma.Multiaddr) bool {
		first, _ := ma.SplitFirst(addr)
		if first == nil {
			return false
		}
		if first.Protocol().Code != ma.P_IP4 {
			return false
		}
		return isQUICAddr(addr)
	})
}

func (f *quicAddrFilter) FilterRemote(remoteID peer.ID, maddrs []ma.Multiaddr) []ma.Multiaddr {
	return f.filterQUICIPv4(remoteID, maddrs)
}

func (f *quicAddrFilter) FilterLocal(remoteID peer.ID, maddrs []ma.Multiaddr) []ma.Multiaddr {
	return f.filterQUICIPv4(remoteID, maddrs)
}

func generateSelfSignedCert(priv ed25519.PrivateKey, dnsNames []string) (*tls.Certificate, error) {
	// Generate a new ed25519 key pair
	pub := priv.Public().(ed25519.PublicKey)

	// Create a certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),           // Valid from 1 hour ago
		NotAfter:     time.Now().Add(24 * time.Hour * 365), // Valid for 1 year
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, err
	}

	// Create the tls.Certificate
	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	return cert, nil
}

func isQUICAddr(a ma.Multiaddr) bool {
	return mafmt.And(mafmt.IP, mafmt.Base(ma.P_UDP), mafmt.Base(ma.P_QUIC_V1)).Matches(a)
}

func quicAddrToNetAddr(a ma.Multiaddr) (*net.UDPAddr, error) {
	first, _ := ma.SplitFunc(a, func(c ma.Component) bool { return c.Protocol().Code == ma.P_QUIC_V1 })
	if first == nil {
		return nil, fmt.Errorf("no QUIC address found in multiaddr")
	}
	netAddr, err := manet.ToNetAddr(first)
	if err != nil {
		return nil, fmt.Errorf("failed to convert multiaddr to net.Addr: %w", err)
	}
	return netAddr.(*net.UDPAddr), nil
}
