# NeoPRISM Integration Test Guide

This guide explains how to run the `TestPRISMNeoPRISMIntegration` test, which tests DID creation and resolution against a local NeoPRISM instance running on a Cardano testnet.

## Prerequisites

1. **Nix development environment**: Enter the dev shell:
   ```bash
   nix develop
   ```

2. **Cardano Node repository**: Clone the Cardano node repository (if not already present):
   ```bash
   # If cardano-node is not already in the workspace
   git clone https://github.com/IntersectMBO/cardano-node.git
   cd cardano-node
   ```

3. **Build cardano-testnet**: Build the `cardano-testnet` binary using Cabal:
   ```bash
   # From the cardano-node directory
   cabal build cardano-testnet
   
   # Find the binary path
   cabal list-bin cardano-testnet
   # Or use the helper script
   ./scripts/bin-path.sh cardano-testnet
   ```
   
   The binary will be in `dist-newstyle/build/.../cardano-testnet/cardano-testnet`. You can add it to your PATH or use the full path.

4. **Cardano testnet environment**: Generate a testnet environment using `cardano-testnet`.

   **Generate the testnet environment:**
   ```bash
   # From the dms directory
   cd builds/docker
   
   # Get the path to cardano-testnet binary
   CARDANO_TESTNET_BIN=$(cd ../../cardano-node && cabal list-bin cardano-testnet 2>/dev/null || echo "cardano-testnet")
   
   # Generate testnet environment (this will also start the node)
   $CARDANO_TESTNET_BIN cardano \
     --conway-era \
     --testnet-magic 42 \
     --active-slots-coeff 0.1 \
     --epoch-length 60 \
     --num-pool-nodes 1 \
     --output-dir testnet-env
   ```
   
   **Or if cardano-testnet is in your PATH:**
   ```bash
   cd builds/docker
   
   cardano-testnet cardano \
     --conway-era \
     --testnet-magic 42 \
     --active-slots-coeff 0.1 \
     --epoch-length 60 \
     --num-pool-nodes 1 \
     --output-dir testnet-env
   ```
   
   This command will:
   - Create a testnet environment at `builds/docker/testnet-env/` containing:
     - Genesis files (`byron-genesis.json`, `shelley-genesis.json`, `alonzo-genesis.json`, `conway-genesis.json`)
     - Node configuration (`configuration.yaml`)
     - Node socket (`socket/node1/sock`)
     - UTXO keys for funding wallets (`utxo-keys/utxo1/`, `utxo-keys/utxo2/`, etc.)
     - Stake pool keys (`pools-keys/pool1/`, etc.)
     - Delegate keys, genesis keys, and other testnet keys
   - **Automatically start the cardano-node** that uses this environment
   - The node will run in the foreground and produce blocks
   
   **Verify the testnet is running:**
   ```bash
   # In another terminal, check that the node is producing blocks
   cardano-cli query tip \
     --testnet-magic 42 \
     --socket-path builds/docker/testnet-env/socket/node1/sock
   ```
   
   You should see output showing the current block height and slot number.
   
   **Important Notes**: 
   - If the testnet environment already exists, you can skip this step and use the existing one
   - The `cardano-testnet` command will automatically start the node when generating the testnet environment
   - The node runs in the foreground - keep this terminal open or run it in a screen/tmux session
   - To stop the node, press `Ctrl+C` or kill the process
   - The testnet environment is persistent and can be reused across test runs
   - The node must be running for NeoPRISM to connect to it

## Step 1: Start NeoPRISM Services

Start the Docker Compose stack that includes NeoPRISM, Cardano DB Sync, and Cardano Wallet:

```bash
cd builds/docker

# Start all services
docker compose -f prism-cardano-testnet.yml up -d

# Check service status
docker compose -f prism-cardano-testnet.yml ps

# View logs (optional)
docker compose -f prism-cardano-testnet.yml logs -f neoprism
```

The compose file is configured to:
- Use the external cardano-node socket at `builds/docker/testnet-env/socket/node1/sock`
- Mount genesis files from `builds/docker/testnet-env/`
- Start NeoPRISM on port `8080`
- Start Cardano Wallet on port `8091`
- Start Cardano DB Sync to index blockchain data

### Wait for Services to be Ready

The services need time to initialize:

1. **Cardano DB Sync**: Wait for it to sync with the blockchain (this can take several minutes)
   ```bash
   docker compose -f prism-cardano-testnet.yml logs -f cardano-db-sync
   ```
   Look for messages indicating successful sync.

2. **NeoPRISM**: Wait for it to start and connect to DB Sync
   ```bash
   docker compose -f prism-cardano-testnet.yml logs -f neoprism
   ```
   Look for "NeoPRISM started" or similar messages.

3. **Cardano Wallet**: Ensure it's connected to the node
   ```bash
   curl http://localhost:8091/v2/network/information
   ```

## Step 2: Create and Fund a Wallet (if needed)

If you need to create a wallet for NeoPRISM to use for submitting transactions:

### Using the Automated Script (Recommended)

Use the provided script that auto-generates a mnemonic using `cardano-wallet` CLI:

```bash
cd builds/docker

# Create wallet with auto-generated mnemonic
./create-neoprism-wallet.sh [wallet-name] [passphrase]

# Example:
./create-neoprism-wallet.sh neoprism-wallet buildb3tt3rNeoprism
```

The script will:
- Generate a recovery phrase (mnemonic) using `cardano-wallet recovery-phrase generate`
- Create the wallet via the cardano-wallet API
- Wait for the wallet to be ready
- Get the first payment address
- Output all environment variables needed for NeoPRISM configuration

**Important**: Save the recovery phrase that the script outputs - you'll need it to restore the wallet.

### Manual Wallet Creation (Alternative)

If you prefer to create the wallet manually:

```bash
# Generate a mnemonic using cardano-wallet CLI
MNEMONIC=$(cardano-wallet recovery-phrase generate)

# Convert to JSON array format
MNEMONIC_JSON=$(echo "$MNEMONIC" | jq -R -s -c 'split("\n") | map(select(length > 0))')

# Create wallet via API
curl -X POST http://localhost:8091/v2/wallets \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"neoprism-wallet\",
    \"mnemonic_sentence\": ${MNEMONIC_JSON},
    \"passphrase\": \"your-passphrase\"
  }"

# Get the wallet ID and payment address
WALLET_ID=$(curl -s http://localhost:8091/v2/wallets | jq -r '.[0].id')
PAYMENT_ADDR=$(curl -s http://localhost:8091/v2/wallets/$WALLET_ID/addresses | jq -r '.[0].id')
```

### Update Environment Variables

After creating the wallet, update the compose file environment variables or set them when starting:

```bash
WALLET_ID=your-wallet-id \
WALLET_PASSPHRASE=your-passphrase \
PAYMENT_ADDRESS=addr_test1... \
docker compose -f prism-cardano-testnet.yml up -d
```

### Fund the Wallet

Fund the wallet using UTXO keys from `testnet-env/utxo-keys/`:

```bash
# Use the fund-wallet.sh script
./fund-wallet.sh ${PAYMENT_ADDR} 100000000
```

## Step 3: Run the Integration Test

Set the environment variable and run the test:

```bash
# From the dms directory
export PRISM_TESTNET_URL=http://localhost:8080

# Run the test
go test -v -run TestPRISMNeoPRISMIntegration ./lib/ucan

# Or with timeout (test may take 20+ minutes due to indexing)
go test -v -timeout 30m -run TestPRISMNeoPRISMIntegration ./lib/ucan
```

## What the Test Does

1. **Generates a Secp256k1 key pair** (required by NeoPRISM for master keys)
2. **Creates a signed PRISM operation** using `did.CreateSignedPRISMOperationSimple`
3. **Submits the operation** to NeoPRISM's `/api/signed-operation-submissions` endpoint
4. **Waits for the DID to be indexed** (can take 10-20 minutes depending on indexer position)
5. **Resolves the DID document** from NeoPRISM's `/api/dids/{did}` endpoint
6. **Verifies the DID document** contains verification methods
7. **Tests UCAN functionality**:
   - Creates a DID provider from the resolved DID
   - Signs and verifies messages
   - Creates trust and capability contexts
   - Grants and verifies UCAN tokens
   - Tests capability delegation

## Troubleshooting

### Test Times Out Waiting for DID

The test waits for the DID to be indexed by NeoPRISM. This can take 10-20 minutes if:
- The indexer is far behind the current block
- The transaction was submitted in a recent block

**Solution**: Check the indexer position:
```bash
curl http://localhost:8080/api/indexer-stats | jq
```

Compare the `last_prism_block_number` with the block where your transaction was submitted.

### DID Resolution Returns 404

This usually means:
1. The DID hasn't been indexed yet (wait longer)
2. The transaction wasn't confirmed on the blockchain
3. DB Sync hasn't synced the transaction yet

**Solution**: 
- Check transaction confirmation: query the transaction hash in DB Sync
- Check indexer progress: `curl http://localhost:8080/api/indexer-stats`
- Check NeoPRISM logs: `docker compose -f prism-cardano-testnet.yml logs neoprism`

### NeoPRISM Can't Connect to DB Sync

Check that:
1. DB Sync is running: `docker compose -f prism-cardano-testnet.yml ps db-sync`
2. DB Sync has synced: check logs for sync progress
3. The database name is correct: should be `cexplorer` (not `cexplore`)

### Cardano Wallet Not Ready

If NeoPRISM can't submit transactions:
1. Check wallet is running: `curl http://localhost:8091/v2/network/information`
2. Check wallet has funds: query the payment address
3. Verify wallet ID and passphrase are correct in compose file

## Cleaning Up

To stop all services:

```bash
cd builds/docker

# Stop compose services
docker compose -f prism-cardano-testnet.yml down

# Stop cardano-node (if running in Docker)
docker stop cardano-testnet-node
docker rm cardano-testnet-node

# Or if running directly, kill the process
pkill -f "cardano-node run"
```

To remove all data (start fresh):

```bash
cd builds/docker

# Remove Docker volumes
docker compose -f prism-cardano-testnet.yml down -v

# Remove testnet environment (if you want to regenerate)
# rm -rf testnet-env/
```

## Notes

- The testnet environment (`builds/docker/testnet-env/`) is persistent and can be reused across test runs
- NeoPRISM requires **Secp256k1 keys** for master keys (not Ed25519)
- The test includes retry logic with exponential backoff for DID resolution
- Indexing can take significant time; the test has a long timeout (default 20+ minutes)
- The compose file uses absolute paths for the testnet socket; adjust if your setup differs

