package libp2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"gitlab.com/nunet/device-management-service/models"
)

func (p2p Libp2p) BootstrapNode(ctx context.Context) error {
	return Bootstrap(ctx, p2p.Host, p2p.DHT)
}

func Bootstrap(ctx context.Context, node host.Host, idht *dht.IpfsDHT) error {
	if err := idht.Bootstrap(ctx); err != nil {
		return err
	}

	for _, nb := range NuNetBootstrapPeers {
		p, _ := peer.AddrInfoFromP2pAddr(nb)
		if err := node.Connect(ctx, *p); err != nil {
			zlog.Sugar().Errorf("failed to connect to bootstrap node %s - %v", p.ID.String(), err)
		} else {
			zlog.Sugar().Infof("Connected to Bootstrap Node %s", p.ID.String())
		}
	}

	zlog.Info("Done Bootstrapping")
	return nil
}

// Cleans up offline peers from DHT
func (p Libp2p) CleanupOfflinePeers() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	for _, node := range p.Host.Peerstore().Peers() {
		peerData, err := p.Host.Peerstore().Get(node, "peer_info")
		if err != nil {
			continue
		}
		if node == p.Host.ID() {
			continue
		}
		if Data, ok := peerData.(models.PeerData); ok {
			targetPeer, err := peer.Decode(Data.PeerID)
			if err != nil {
				zlog.Sugar().Errorf("Error decoding peer ID: %v", err)
				continue
			}
			pingResult, pingCancel := p.Ping(ctx, targetPeer)
			result := <-pingResult
			if result.Error == nil {
				if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
					zlog.Sugar().Infof("Peer is reachable. PeerID: %s", Data.PeerID)
					pingCancel()
					continue
				}
			} else {
				if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
					zlog.Sugar().Infof("Peer - %s is unreachable. Removing from Peerstore.", Data.PeerID)
					p.Host.Peerstore().Put(node, "peer_info", nil)
				}
			}
			pingCancel()
		}
	}
}

func (p2p Libp2p) fetchKadDhtContents(ctxt context.Context, resultChan chan models.PeerData) {
	zlog.Debug("Fetching DHT content for all peers")

	fetchCtx, _ := context.WithTimeout(ctxt, time.Minute)

	go func() {
		// Create a wait group to ensure all workers have finished
		var wg sync.WaitGroup

		// Create a buffered channel for the worker pool
		poolSize := 5 // Adjust the pool size as per your requirements
		workerPool := make(chan struct{}, poolSize)

		zlog.Sugar().Debugf("FetchKadDHTContents: Starting workers - number of p2p.peers: %d", len(p2p.peers))

		for _, p := range p2p.peers {
			zlog.Sugar().Debugf("FetchKadDHTContents: Waiting for worker slot for peer: %s", p.ID.String())
			workerPool <- struct{}{} // Acquire a worker slot from the pool
			zlog.Sugar().Debugf("FetchKadDHTContents: Acquired worker slot for peer: %s", p.ID.String())
			wg.Add(1) // Increment the wait group counter

			zlog.Sugar().Debugf("FetchKadDHTContents: Fetching DHT content for peer: %s ", p.ID.String())
			go func(peer peer.AddrInfo) {
				defer func() {
					<-workerPool // Release the worker slot
					wg.Done()    // Signal the wait group that the worker is done
					zlog.Sugar().Debugf("FetchKadDHTContents: Worker for %s finished", peer.ID.String())
				}()

				var updates models.KadDHTMachineUpdate

				// Add custom namespace to the key
				namespacedKey := customNamespace + peer.ID.String()
				bytes, err := p2p.DHT.GetValue(fetchCtx, namespacedKey)

				if err != nil {
					if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
						zlog.Sugar().Errorf(fmt.Sprintf("Couldn't retrieve dht content for peer: %s", peer.ID.String()))
					}
					return
				}

				err = json.Unmarshal(bytes, &updates)
				if err != nil {
					if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
						zlog.Sugar().Errorf("Error unmarshalling value: %v", err)
					}
					return
				}

				peerInfo := models.PeerData{}
				err = json.Unmarshal(updates.Data, &peerInfo)
				if err != nil {
					if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
						zlog.Sugar().Errorf("Error unmarshalling value: %v", err)
					}
					return
				}

				// Send the fetched value through the result channel
				resultChan <- peerInfo
			}(p)
		}

		zlog.Debug("FetchKadDHTContents: Waiting for workers to finish")
		wg.Wait()
		zlog.Debug("FetchKadDHTContents: All workers Done. Closing channel")
		close(resultChan)
	}()
}

// Fetches peer info of peers from Kad-DHT and updates Peerstore.
func (p Libp2p) GetDHTUpdates(ctx context.Context) {
	if gettingDHTUpdate {
		zlog.Debug("GetDHTUpdates: Already Getting DHT Updates")
		return
	}
	gettingDHTUpdate = true
	zlog.Debug("GetDHTUpdates: Start Getting DHT Updates")

	machines := make(chan models.PeerData)
	p.fetchKadDhtContents(ctx, machines)

	for machine := range machines {
		zlog.Sugar().Debugf("GetDHTUpdates: Got machine: %v", machine.PeerID)
		targetPeer, err := peer.Decode(machine.PeerID)
		if err != nil {
			zlog.Sugar().Errorf("Error decoding peer ID: %v", err)
			gettingDHTUpdate = false
			continue
		}
		pingResult, pingCancel := p.Ping(ctx, targetPeer)
		res := <-pingResult
		if res.Error == nil {
			if _, verboseDebugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); verboseDebugMode {
				zlog.Sugar().Info("Peer is reachable.", "PeerID", machine.PeerID)
			}
			err := p.Host.Peerstore().Put(targetPeer, "peer_info", machine)
			if err != nil {
				zlog.Sugar().Errorf("Error putting peer info of %s in peerstore: %v", targetPeer.String(), err)
			}
		} else {
			if _, verboseDebugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); verboseDebugMode {
				zlog.Sugar().Info("Peer -  ", machine.PeerID, " is unreachable.")
			}
		}
		pingCancel()
	}
	gettingDHTUpdate = false
	doneGettingDHTUpdate <- true
	zlog.Debug("Done Getting DHT Updates")
}

func signData(hostPrivateKey crypto.PrivKey, data []byte) ([]byte, error) {
	signature, err := hostPrivateKey.Sign(data)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

type dhtValidator struct {
	PS peerstore.Peerstore
}

func (d dhtValidator) Validate(key string, value []byte) error {
	// Check if the key has the correct namespace
	if !strings.HasPrefix(key, customNamespace) {
		return errors.New("invalid key namespace")
	}

	components := strings.Split(key, "/")
	key = components[len(components)-1]
	var dhtUpdate models.KadDHTMachineUpdate

	err := json.Unmarshal(value, &dhtUpdate)
	if err != nil {
		zlog.Sugar().Errorf("Error unmarshalling value: %v", err)
		return err
	}

	// Extract data and signature fields
	data := dhtUpdate.Data
	var peerInfo models.PeerData
	err = json.Unmarshal(dhtUpdate.Data, &peerInfo)
	if err != nil {
		zlog.Sugar().Errorf("Error unmarshalling value: %v", err)
		return err
	}

	signature := dhtUpdate.Signature
	remotePeerID, err := peer.Decode(key)
	if err != nil {
		zlog.Sugar().Errorf("Error decoding peerID: %v", err)
		return errors.New("error decoding peerID")
	}

	// Get the public key of the remote peer from the peerstore
	remotePeerPublicKey := d.PS.PubKey(remotePeerID)
	if remotePeerPublicKey == nil {
		return errors.New("public key for remote peer not found in peerstore")
	}
	verify, err := remotePeerPublicKey.Verify(data, signature)
	if err != nil {
		zlog.Sugar().Errorf("Error verifying signature: %v", err)
		return err
	}
	if !verify {
		zlog.Sugar().Info("Invalid signature")
		return errors.New("invalid signature")
	}

	if len(value) == 0 {
		return errors.New("value cannot be empty")
	}
	return nil
}
func (dhtValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

// TODO remove the below when network package is fully implemented
// UpdateKadDHT is a stub
func (p Libp2p) UpdateKadDHT() {
	zlog.Warn("UpdateKadDHT: Stub")
}

// ListKadDHTPeers is a stub
func (p Libp2p) ListKadDHTPeers(c *gin.Context, ctx context.Context) ([]string, error) {
	zlog.Warn("ListKadDHTPeers: Stub")
	return nil, nil
}

// SelfPeer is a stub - peer.AddrInfo should replace any usage
type SelfPeer struct {
	ID    string
	Addrs []multiaddr.Multiaddr
}

// SelfPeerInfo is a stub
func (p Libp2p) SelfPeerInfo() (*SelfPeer, error) {
	zlog.Warn("SelfPeerInfo: Stub")
	return nil, nil
}
