> For an interactive script to assist with this setup, check it [here](../../maint-scripts/private_network.sh).

# Creating a Private Network

By default, in NuNet's p2p network, all nodes share the same underlying network infrastructure for communication.

However, the capability system acts as an access control layer that determines which peers can invoke
specific behaviors (e.g.: deploying an allocation) on other peers.

When following the main README usage guide, you're connecting to NuNet's official network where the capability pool is built upon KYC-verified peers.
NuNet acts as the root of trust, issuing capability tokens to verified participants.

This guide demonstrates how any organization or entity can create their own private network which is independent
from NuNet Foundation's trust pool.

> If you want to be able to participate on both NuNet and any other private networks at once, as an user, you just have
> to set up the capabilities for each root entity controlling the capability pool.
>
> In practice, you just have to follow the user guide side of this documentation and also the guide on DMS readme.

## Overview

Creating a private network involves:

1. Setting up an organization key/context that acts as the root of trust
2. With the organization key/context, grant capabilities and send the generated token to each user

This guide consider users having a key/context for one user and `n` dmses.

Also, let's suppose an organization named `myorg`.

## Step-by-Step Guide: myorg side

### 1. (myorg) Create Organization Key/Context

First, create a key and capability context for your organization:

```bash
nunet key new myorg
# Save the returned DID as <did-myorg>
nunet cap new myorg
```

> **Important**: Securely store your organization's private key

### 2. (myorg) Grant Organization Capabilities to each user

Grant each user capabilities to invoke certain behaviors:

```bash
nunet cap grant --context myorg --cap /dms/deployment --cap /broadcast --cap /public --topic /nunet --expiry 2024-12-31 <did-user>
# Save the returned token as <token-1>
```

Send the generated `<token-1>` to user of did `<did-user>`.

## Step-by-Step Guide: user side

Considering an user with keys/contexts for both `user` and `dms`:

### 1. (user) Anchor token sent by myorg

```bash
nunet cap anchor --context user --provide <token-1>
```

### 2. (user) Set Up DMS Anchors

Grant and set up the necessary require and provide anchors for user's DMS:

```bash
# Grant from user to organization (for require anchor)
nunet cap grant --context user --cap /dms/deployment --cap /broadcast --cap /public --topic /nunet --expiry 2024-12-31 <did-myorg>
# Save the returned token as <token-2>

# Add require anchor to DMS
nunet cap anchor --context dms --require <token-2>

# Delegate from user to DMS (for provide anchor)
nunet cap delegate --context user --cap /dms/deployment --cap /broadcast --cap /public --topic /nunet --expiry 2024-12-31 <did-dms>
# Save the returned token as <token-3>

# Add provide anchor to DMS
nunet cap anchor --context dms --provide <token-3>
```

## Optional: use custom bootstrap nodes

Following the previous guide, you successfully built your own capability pool:

Nodes will only be able to deploy allocations in nodes that have been granted the `/dms/deployment` capability by your organization.

Though, as explained in the introduction, your nodes would still share the same underlying network. When broadcasting bid requests for
a deployment, all peers in NuNet network would receive the request, even peers outside your capability pool.

To enhance privacy, security and scability of your private network, you can use your own bootstrap nodes instead of NuNet's.

For that, you have to customize your `dms_config.json`. This is how your `bootstrap_peers` section
looks now:

```json
...
"p2p": {
    "bootstrap_peers": [
      "/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmQ2irHa8aFTLRhkbkQCRrounE4MbttNp8ki7Nmys4F9NP",
      "/dnsaddr/bootstrap.p2p.nunet.io/p2p/Qmf16N2ecJVWufa29XKLNyiBxKWqVPNZXjbL3JisPcGqTw",
      "/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ"
    ],
...
}
...
```

To edit your `dms_config.json`, run:

```bash
nunet config edit
```

Modify the bootstrap peers to use your own nodes.

> **Multiaddresses:**
> If using a domain name: `/dnsaddr/bootstrap.p2p.nunet.io/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ`
> If using an IP address: `/ip4/<IP_ADDRESS>/p2p/QmTkWP72uECwCsiiYDpCFeTrVeUM9huGTPsg3m6bHxYQFZ`

`Qm...` is the bootstrap's peer ID.

To retrieve a peer ID from a given node, run:

```bash
nunet actor cmd /dms/node/peers/self -c <your_user_context>
```

## Key Differences from NuNet Trusted Network

1. **Root of Trust**: Your organization key becomes the root of trust instead of NuNet
2. **Token Distribution**: You control token distribution and can add/revoke access as needed
