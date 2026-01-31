#!/bin/bash
# Create a wallet for NeoPRISM using cardano-wallet CLI
# This script auto-generates a mnemonic and creates a wallet, then outputs
# the environment variables needed for NeoPRISM configuration.
#
# Usage: ./create-neoprism-wallet.sh [wallet-name] [passphrase]
#
# Prerequisites:
#   - cardano-wallet container must be running (container name: cardano-wallet)
#   - cardano-wallet API must be accessible
#   - docker must be installed and cardano-wallet container must exist
#   - jq must be installed

set -e

WALLET_NAME="${1:-neoprism-wallet}"
WALLET_PASSPHRASE="${2:-buildb3tt3rNeoprism}"
CARDANO_WALLET_API="${CARDANO_WALLET_API:-http://localhost:8091}"
CARDANO_WALLET_CONTAINER="${CARDANO_WALLET_CONTAINER:-cardano-wallet}"

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Error: docker not found. Please install docker."
    exit 1
fi

# Check if cardano-wallet container exists and is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CARDANO_WALLET_CONTAINER}$"; then
    echo "❌ Error: cardano-wallet container '${CARDANO_WALLET_CONTAINER}' is not running."
    echo "   Make sure the container is started with: docker-compose up -d cardano-wallet"
    exit 1
fi

# Function to run cardano-wallet CLI commands in the container
cardano_wallet_cli() {
    docker exec "${CARDANO_WALLET_CONTAINER}" cardano-wallet "$@"
}

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "❌ Error: jq not found. Please install jq."
    exit 1
fi

# Check if cardano-wallet API is accessible
if ! curl -s -f "${CARDANO_WALLET_API}/v2/network/information" > /dev/null; then
    echo "❌ Error: Cannot connect to cardano-wallet at ${CARDANO_WALLET_API}"
    echo "   Make sure cardano-wallet is running and accessible."
    exit 1
fi

echo "🔧 Creating wallet: $WALLET_NAME"
echo ""

# Generate a mnemonic using cardano-wallet CLI
# cardano-wallet recovery-phrase generate creates a 15-word mnemonic by default
echo "📝 Generating recovery phrase (mnemonic)..."
MNEMONIC=$(cardano_wallet_cli recovery-phrase generate)

if [ -z "$MNEMONIC" ]; then
    echo "❌ Error: Failed to generate mnemonic"
    exit 1
fi

# Convert mnemonic to JSON array format for API
# Split by whitespace and create array with each word as separate element
MNEMONIC_JSON=$(echo "$MNEMONIC" | jq -R -c 'split(" ") | map(select(. != ""))')

echo "✅ Generated mnemonic (saved for reference)"
echo ""

echo "MNEMONIC_JSON: $MNEMONIC_JSON"

# Create wallet via API using the generated mnemonic
echo "💼 Creating wallet via cardano-wallet API..."
WALLET_RESPONSE=$(curl -s -X POST "${CARDANO_WALLET_API}/v2/wallets" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"${WALLET_NAME}\",
        \"mnemonic_sentence\": ${MNEMONIC_JSON},
        \"passphrase\": \"${WALLET_PASSPHRASE}\"
    }")

# Check if wallet creation was successful
if echo "$WALLET_RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    ERROR_MSG=$(echo "$WALLET_RESPONSE" | jq -r '.message // .error')
    echo "❌ Error creating wallet: $ERROR_MSG"
    exit 1
fi

WALLET_ID=$(echo "$WALLET_RESPONSE" | jq -r '.id')

if [ -z "$WALLET_ID" ] || [ "$WALLET_ID" = "null" ]; then
    echo "❌ Error: Failed to create wallet. Response:"
    echo "$WALLET_RESPONSE" | jq '.'
    exit 1
fi

echo "✅ Wallet created: $WALLET_ID"
echo ""

# Wait for wallet to be ready
echo "⏳ Waiting for wallet to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
WALLET_READY=false

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    WALLET_STATUS=$(curl -s "${CARDANO_WALLET_API}/v2/wallets/${WALLET_ID}" | jq -r '.state.state // "unknown"')
    
    if [ "$WALLET_STATUS" = "ready" ]; then
        WALLET_READY=true
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    sleep 1
done

if [ "$WALLET_READY" = false ]; then
    echo "⚠️  Warning: Wallet may not be ready yet. Current state: $WALLET_STATUS"
    echo "   You may need to wait a bit longer for the wallet to sync."
else
    echo "✅ Wallet is ready"
fi
echo ""

# Get the first payment address
echo "📍 Getting payment address..."
ADDRESSES_RESPONSE=$(curl -s "${CARDANO_WALLET_API}/v2/wallets/${WALLET_ID}/addresses")

if [ "$(echo "$ADDRESSES_RESPONSE" | jq 'length')" -eq 0 ]; then
    # Create a new address if none exist
    echo "   No addresses found, creating new address..."
    NEW_ADDRESS_RESPONSE=$(curl -s -X POST "${CARDANO_WALLET_API}/v2/wallets/${WALLET_ID}/addresses" \
        -H "Content-Type: application/json" \
        -d '{}')
    PAYMENT_ADDR=$(echo "$NEW_ADDRESS_RESPONSE" | jq -r '.id')
else
    PAYMENT_ADDR=$(echo "$ADDRESSES_RESPONSE" | jq -r '.[0].id')
fi

if [ -z "$PAYMENT_ADDR" ] || [ "$PAYMENT_ADDR" = "null" ]; then
    echo "❌ Error: Failed to get payment address"
    exit 1
fi

echo "✅ Payment address: $PAYMENT_ADDR"
echo ""

# Output environment variables
echo "========================================="
echo "  NeoPRISM Wallet Configuration"
echo "========================================="
echo ""
echo "# Add these to your .env file or docker-compose.yml:"
echo ""
echo "WALLET_BASE_URL=${CARDANO_WALLET_API}"
echo "WALLET_ID=${WALLET_ID}"
echo "WALLET_PASSPHRASE=${WALLET_PASSPHRASE}"
echo "PAYMENT_ADDRESS=${PAYMENT_ADDR}"
echo ""
echo "# For NeoPRISM environment variables:"
echo ""
echo "NPRISM_CARDANO_WALLET_BASE_URL=${CARDANO_WALLET_API}/v2"
echo "NPRISM_CARDANO_WALLET_WALLET_ID=${WALLET_ID}"
echo "NPRISM_CARDANO_WALLET_PASSPHRASE=${WALLET_PASSPHRASE}"
echo "NPRISM_CARDANO_WALLET_PAYMENT_ADDR=${PAYMENT_ADDR}"
echo ""
echo "========================================="
echo ""
echo "⚠️  IMPORTANT: Save your recovery phrase in a secure location:"
echo ""
echo "$MNEMONIC" | sed 's/^/   /'
echo ""
echo "========================================="
echo ""
echo "💡 To fund this wallet, use:"
echo "   ./fund-wallet.sh ${PAYMENT_ADDR} 100000000"
echo "   (or use UTXO keys from testnet-env/utxo-keys/)"
echo ""

