package orchestrator

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	netutils "gitlab.com/nunet/device-management-service/network/utils"
)

var orchestratorJoinTimeout = 2 * time.Minute

type SubnetManifest struct {
	CIDR              string            `json:"cidr"`
	GatewayIP         string            `json:"gateway_ip"`
	BroadcastIP       string            `json:"broadcast_ip"`
	UsedIPs           map[string]bool   `json:"used_ips"`
	RoutingTable      map[string]string `json:"routing_table"`
	IndexRoutingTable map[string]string `json:"index_routing_table"`
}

type subnetRequest struct {
	handle actor.Handle
	ip     string
	peerID string
	ports  map[int]int
}

type SubnetCreateRequest struct {
	SubnetID     string
	IP           string
	RoutingTable map[string]string
}

type SubnetCreateResponse struct {
	OK    bool
	Error string
}

type SubnetJoinRequest struct {
	SubnetID string
	PeerID   string
	IP       string

	// map of domain_name:ip
	Records map[string]string
}

type SubnetJoinResponse struct {
	OK    bool
	Error string
}

func newSubnetManifest() (SubnetManifest, error) {
	cidr, err := netutils.GetRandomCIDRInRange(
		24,
		net.ParseIP("10.0.0.0"),
		net.ParseIP("10.255.255.255"),
		[]string{},
	)
	if err != nil {
		return SubnetManifest{}, fmt.Errorf("error getting random CIDR: %w", err)
	}

	parts := strings.Split(strings.Split(cidr, "/")[0], ".")
	gatewayIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "1")
	broadcastIP := fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], "255")
	usedIPs := map[string]bool{
		gatewayIP:   true,
		broadcastIP: true,
	}

	return SubnetManifest{
		CIDR:              cidr,
		GatewayIP:         gatewayIP,
		BroadcastIP:       broadcastIP,
		UsedIPs:           usedIPs,
		RoutingTable:      make(map[string]string),
		IndexRoutingTable: make(map[string]string),
	}, nil
}

func (p *Provisioner) createSubnet(
	manifestID string,
	subReqs []subnetRequest, routingTable map[string]string,
	subCreateHandles []actor.Handle,
) error {
	errCh := make(chan error, len(subReqs))
	wg := sync.WaitGroup{}

	for _, handle := range subCreateHandles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				handle,
				fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, manifestID),
				SubnetCreateRequest{
					SubnetID:     manifestID,
					RoutingTable: routingTable,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response SubnetCreateResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet response: %w", err)
					return
				}
				if !response.OK {
					errCh <- fmt.Errorf("error creating subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(SubnetCreateTimeout):
				errCh <- fmt.Errorf("timeout creating subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("subnet successfully created on peer", handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

// TODO: this step should be together with subnet creation, we have
// to refactor the SubnetCreate handle
func (p *Provisioner) subnetAddPeer(manifestID string, subReqs []subnetRequest) error {
	// 1.b create and plug IPs
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				req.handle,
				behaviors.SubnetAddPeerBehavior,
				behaviors.SubnetAddPeerRequest{
					SubnetID: manifestID,
					IP:       req.ip,
					PeerID:   req.peerID,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-peer message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-peer message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetAddPeerResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}
			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout adding peer to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("peer successfully added to subnet on peer", req.handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) addDNSRecords(
	manifestID string,
	subReqs []subnetRequest, dnsRecords map[string]string,
) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))

	for _, req := range subReqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := actor.Message(
				p.actor.Handle(),
				req.handle,
				behaviors.SubnetDNSAddRecordsBehavior,
				behaviors.SubnetDNSAddRecordsRequest{
					SubnetID: manifestID,
					Records:  dnsRecords,
				},
				actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
			)
			if err != nil {
				errCh <- fmt.Errorf("error creating subnet add-dns-records message: %w", err)
				return
			}

			replyCh, err := p.actor.Invoke(msg)
			if err != nil {
				errCh <- fmt.Errorf("error invoking subnet add-dns-records message: %w", err)
				return
			}

			var reply actor.Envelope
			select {
			case reply = <-replyCh:
				defer reply.Discard()

				var response behaviors.SubnetDNSAddRecordsResponse
				if err := json.Unmarshal(reply.Message, &response); err != nil {
					errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
					return
				}

				if !response.OK {
					errCh <- fmt.Errorf("error sending dns records to peer: %s: %w", response.Error, ErrDeploymentFailed)
					return
				}

			case <-time.After(2 * time.Minute):
				errCh <- fmt.Errorf("timeout sending dns records to subnet: %w", ErrDeploymentFailed)
				return
			}

			log.Info("DNS records successfully added to subnet on peer", req.handle)
		}()
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

func (p *Provisioner) mapPorts(manifestID string, subReqs []subnetRequest) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, len(subReqs))
	for _, req := range subReqs {
		for pubPort := range req.ports {
			wg.Add(1)
			go func() {
				defer wg.Done()
				msg, err := actor.Message(
					p.actor.Handle(),
					req.handle,
					behaviors.SubnetMapPortBehavior,
					behaviors.SubnetMapPortRequest{
						SubnetID:   manifestID,
						Protocol:   "TCP", // TODO: add support in AllocationManifest for protocol
						SourceIP:   "0.0.0.0",
						SourcePort: strconv.Itoa(pubPort),
						DestIP:     req.ip,
						DestPort:   strconv.Itoa(pubPort),
					},
					actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
				)
				if err != nil {
					errCh <- fmt.Errorf("error creating subnet MapPort message: %w", err)
					return
				}

				replyCh, err := p.actor.Invoke(msg)
				if err != nil {
					errCh <- fmt.Errorf("error invoking subnet MapPort message: %w", err)
					return
				}

				var reply actor.Envelope
				select {
				case reply = <-replyCh:
					defer reply.Discard()

					var response behaviors.SubnetMapPortResponse
					if err := json.Unmarshal(reply.Message, &response); err != nil {
						errCh <- fmt.Errorf("error unmarshalling subnet add-peer response: %w", err)
						return
					}

					if !response.OK {
						errCh <- fmt.Errorf("error adding peer to subnet: %s: %w", response.Error, ErrDeploymentFailed)
						return
					}
				case <-time.After(2 * time.Minute):
					errCh <- fmt.Errorf("timeout mapping port for subnet: %w", ErrDeploymentFailed)
					return
				}

				log.Info("port mapping successfully added to subnet on peer", req.handle)
			}()
		}
	}

	wg.Wait()
	close(errCh)

	return aggregateErrors(errCh)
}

// TODO: maybe this hsould go to the createSubnet method
func (p *Provisioner) orchestratorJoinSubnet(
	manifestID string,
	indexRoutingTable map[string]string, dnsRecords map[string]string,
) error {
	msg, err := actor.Message(
		p.actor.Handle(),
		p.actor.Supervisor(),
		fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, manifestID),
		SubnetJoinRequest{
			SubnetID: manifestID,
			IP:       indexRoutingTable[orchSubnetName],
			PeerID:   p.actor.Handle().Address.HostID,
			Records:  dnsRecords,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
	)
	if err != nil {
		return fmt.Errorf("error creating subnet join message: %w", err)
	}

	replyCh, err := p.actor.Invoke(msg)
	if err != nil {
		return fmt.Errorf("error invoking subnet join message: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		var response SubnetJoinResponse
		if err := json.Unmarshal(reply.Message, &response); err != nil {
			return fmt.Errorf("error unmarshalling subnet join response: %w", err)
		}

		if !response.OK {
			return fmt.Errorf("error joining orchestrator to subnet: %s: %w", response.Error, ErrDeploymentFailed)
		}
	case <-time.After(orchestratorJoinTimeout):
		return fmt.Errorf("timeout joining orchestrator to subnet: %w", ErrDeploymentFailed)
	}

	log.Info("orchestrator successfully joined the subnet")
	return nil
}

// TODO: maybe use uber multierr
func aggregateErrors(errCh chan error) error {
	var aggErr error
	for err := range errCh {
		if aggErr == nil {
			aggErr = err
			continue
		} else if err != nil {
			aggErr = fmt.Errorf("%w\n%w", aggErr, err)
		}
	}
	return aggErr
}
