# NAT Testing - Quick Reference

## Overview

Tests P2P connectivity through Network Address Translation (NAT) using Incus container-based NAT routers to verify DMS nodes can traverse NAT via libp2p relay and AutoNAT mechanisms.

## NAT Simulation Design

### Architecture

```
┌─────────────────── Incus Host ───────────────────┐
│                                                   │
│  NAT Router 1 (Container)     Relay (Container)  │
│  ┌──────────────┐              ┌──────────┐     │
│  │ eth0: public │              │ public   │     │
│  │ eth1: private├─────┐        │ address  │     │
│  └──────────────┘     │        └──────────┘     │
│         │             │              │           │
│    iptables NAT   Alice (VM)         │           │
│         │         172.16.1.10        │           │
│         │             │              │           │
│  NAT Router 2         │         Bob (VM)         │
│  ┌──────────────┐     │         172.16.2.10     │
│  │ eth0: public │     │              │           │
│  │ eth1: private├─────┴──────────────┘           │
│  └──────────────┘                                │
│         │                                         │
│    iptables NAT                                   │
└───────────────────────────────────────────────────┘
```

### Design Rationale

**Why Container-Based NAT Routers?**
- **True NAT behavior**: Actual iptables NAT, not simulated
- **Isolation**: Each node has its own private network (172.16.x.0/24)
- **No host pollution**: All configuration inside containers
- **AutoNAT compatibility**: Private IPs trigger proper NAT detection

**Two-Layer Firewall Approach:**

**Layer 1 - Host-level iptables** (blocks cross-NAT traffic):
```bash
iptables -I FORWARD -s 172.16.1.0/24 -d 172.16.2.0/24 -j DROP
```
Prevents Incus host kernel from routing between private networks.

**Layer 2 - NAT Router container iptables** (provides NAT + AutoNAT detection):
```bash
# Block inbound NEW (AutoNAT probes fail → NAT detected)
iptables -A FORWARD -i eth0 -o eth1 --ctstate NEW -j DROP

# Allow outbound (Alice/Bob can reach Relay)
iptables -A FORWARD -i eth1 -o eth0 -j ACCEPT

# Symmetrical NAT
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE --random-fully
```

**Symmetrical NAT:**
- Each unique connection gets random external port (`--random-fully`)
- Most restrictive NAT type
- Forces relay usage (hole punching won't work)

## File Structure

```
tests/acceptance/
├── nat_test.go                    # Test entry point
├── features/nat.feature           # BDD scenario
├── steps/nat.go                   # Step implementations
├── utils/
│   ├── nat_router.go              # NAT router management
│   ├── incus.go                   # Incus utilities
│   └── cli.go                     # DMS CLI helpers
└── NAT_TESTING_README.md         # This file
```

## Key Configuration

### NAT Router Setup (`utils/nat_router.go`)

**CreateNATRouterContainer:**
- Creates Ubuntu container with dual NICs (external + internal)
- Configures iptables for symmetrical NAT
- Blocks cross-NAT traffic and inbound NEW connections

**Host Firewall Rules:**
- `AddHostCrossNATBlocking()`: Adds temporary iptables rules
- `RemoveHostCrossNATBlocking()`: Cleans up on test completion

### Test Flow (`features/nat.feature`)

1. **Setup**: Create Alice & Bob behind isolated NAT networks
2. **Direct Test**: Alice → Bob (should FAIL due to NAT)
3. **Relay Setup**: Create public relay, connect Alice & Bob
4. **Wait**: 90 seconds for AutoNAT detection + relay circuits
5. **Verify**: Alice & Bob advertise `/p2p-circuit` addresses
6. **Relay Test**: Alice → Relay → Bob (should SUCCEED)

## Running the Test

### Prerequisites

```bash
# 1. Build DMS binary
go build -o tests/acceptance/builds/dms_linux_amd64 .

# 2. Verify Incus running
incus list
```

### Run Test

```bash
# Requires sudo for host-level iptables
sudo make run-acceptance-TestNAT INSTANCE_TYPE=vm
```

**Test Duration:** ~4-5 minutes

### Expected Output

```
[SETUP] Creating 2 DMS nodes behind container-based NAT routers...
[HOST-FIREWALL] Adding rules to block traffic from 172.16.1.1/24...
[NAT-ROUTER] NAT router configured with symmetrical NAT (--random-fully)
...
[CONNECTION] Alice → Bob FAILED ✅ (NAT isolation working)
...
[RELAY] Alice successfully connected to relay
[RELAY] Bob successfully connected to relay
...
[RELAY ADDRESS] ✓ Relay addresses found
...
[CONNECTION] Alice → Bob via relay SUCCEEDED ✅
```

## Key Commands

### Makefile Targets

```bash
# Run NAT test (recommended)
sudo make run-acceptance-TestNAT INSTANCE_TYPE=vm

# Build test binary with capabilities (experimental)
make build-acceptance-tests
make setcap-acceptance
```

### Manual Test Execution

```bash
cd tests/acceptance
sudo INSTANCE_TYPE=vm go test -test.v -tags=acceptance -test.run "^TestNAT/"
```

## Network Configuration

### Private Networks
- `nat-net-1`: 172.16.1.0/24 (Alice)
- `nat-net-2`: 172.16.2.0/24 (Bob)
- Each network: `ipv4.nat=false` (router handles NAT)

### NAT Routers
- External: incusbr0 (10.x.x.x)
- Internal: 172.16.x.1 (gateway for clients)
- Type: Ubuntu container (not VM)

### DMS Nodes
- Type: VM or Container (set via `INSTANCE_TYPE`)
- Network: Single NIC on private network
- Gateway: NAT router IP (172.16.x.1)

## Cleanup

**Automatic:**
- Host iptables rules removed on test completion
- Incus instances and networks deleted
- No persistent changes to host

**Manual (if test interrupted):**
```bash
# Remove host firewall rules
sudo iptables -L FORWARD -n --line-numbers | grep NUNET-NAT-TEST
sudo iptables -D FORWARD <line-number>

# Delete Incus resources
incus list | grep acc-test | awk '{print $2}' | xargs -I {} incus delete {} --force
incus network list | grep nat-net | awk '{print $2}' | xargs -I {} incus network delete {}
```

## Why Sudo is Required

Modern iptables (nf_tables backend) requires root privileges for:
- Modifying FORWARD chain
- Accessing `/run/xtables.lock`
- Interacting with netfilter kernel modules

The test uses temporary iptables rules that are automatically cleaned up.

## Troubleshooting

### Direct Connection Still Works

**Check host firewall:**
```bash
sudo iptables -L FORWARD -n | grep NUNET-NAT-TEST
# Should see DROP rules for cross-NAT traffic
```

**Check NAT router:**
```bash
incus exec acc-test-nat-router-1 -- iptables -L FORWARD -n
# Should see NEW connection blocking
```

### No Relay Addresses Advertised

**Check AutoNAT detection:**
- Relay must be on different network (incusbr0) than Alice/Bob
- Wait full 90 seconds (60s boot delay + 30s for circuits)
- Verify relay is running: `incus exec acc-test-relay -- pgrep nunet`

### Test Hangs or Times Out

**Check VM networking:**
```bash
incus exec acc-test-alice -- ip route
# Should show: default via 172.16.1.1
```

## Reference

- **Full Documentation**: `NAT_TESTING_GUIDE.md` (detailed troubleshooting)
- **Test Scenario**: `features/nat.feature` (BDD specification)
- **Implementation**: `steps/nat.go` (step definitions)
- **NAT Router Code**: `utils/nat_router.go` (container management)

