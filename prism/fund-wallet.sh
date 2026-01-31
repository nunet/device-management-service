#!/bin/bash
# Fund a wallet on the local cardano-testnet
# Usage: ./fund-wallet.sh <wallet-address> [amount-in-ada]
#
# Prerequisites:
#   - cardano-testnet must be running
#   - cardano-cli must be available
#   - UTXO keys must exist in testnet-env/utxo-keys/

set -e

# Configuration
TESTNET_MAGIC=42
SOCKET_PATH="./testnet-env/socket/node1/sock"
TESTNET_ENV_DIR="./testnet-env"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

echo_success() {
    echo -e "${GREEN}✅${NC} $1"
}

echo_error() {
    echo -e "${RED}❌${NC} $1" >&2
}

echo_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check arguments
WALLET_ADDRESS="${1:-}"
AMOUNT_ADA="${2:-100}"

if [ -z "$WALLET_ADDRESS" ]; then
    echo_error "Usage: $0 <wallet-address> [amount-in-ada]"
    echo ""
    echo "Example:"
    echo "  $0 addr_test1qq66nwgvg6ywry9vw9vuhq6hkjreg9wejq06j2kc73xdl6rcjfucrv7r46anrk9mlw0nclxcpl5j7u9sx36gk3tke79sf8y6ce 100"
    exit 1
fi

# Validate amount
if ! [[ "$AMOUNT_ADA" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
    echo_error "Invalid amount: $AMOUNT_ADA"
    exit 1
fi

AMOUNT_LOVELACE=$(echo "$AMOUNT_ADA * 1000000" | bc | cut -d. -f1)
REQUIRED_AMOUNT=$((AMOUNT_LOVELACE + 500000))  # Add buffer for fees

echo_info "Funding wallet: $WALLET_ADDRESS"
echo_info "Amount: $AMOUNT_ADA ADA ($AMOUNT_LOVELACE lovelace)"
echo ""

# Check if cardano-cli exists
if [ ! -f "$CARDANO_CLI" ]; then
    echo_error "cardano-cli not found at: $CARDANO_CLI"
    echo_info "Trying to find cardano-cli..."
    CARDANO_CLI=$(which cardano-cli 2>/dev/null || echo "")
    if [ -z "$CARDANO_CLI" ]; then
        echo_error "cardano-cli not found in PATH either"
        echo_info "You can find it using: ./scripts/bin-path.sh cardano-cli"
        exit 1
    fi
    echo_info "Using cardano-cli from PATH: $CARDANO_CLI"
fi

# Check if socket exists
if [ ! -S "$SOCKET_PATH" ]; then
    echo_error "Cardano node socket not found at: $SOCKET_PATH"
    echo_info "Make sure cardano-testnet is running"
    exit 1
fi

# Check if testnet-env exists
if [ ! -d "$TESTNET_ENV_DIR" ]; then
    echo_error "Testnet environment not found at: $TESTNET_ENV_DIR"
    echo_info "Make sure cardano-testnet has been started"
    exit 1
fi

# Step 1: Find a UTXO and matching signing key
echo_info "Finding UTXO and signing key..."
UTXO_SKEY=""
UTXO_VKEY=""
SOURCE_ADDRESS=""

# Strategy: For each UTXO key, check if it has a UTXO with sufficient funds
for utxo_dir in "$TESTNET_ENV_DIR/utxo-keys/utxo"*; do
    if [ ! -d "$utxo_dir" ]; then
        continue
    fi
    
    skey_file="$utxo_dir/utxo.skey"
    vkey_file="$utxo_dir/utxo.vkey"
    
    if [ ! -f "$skey_file" ] || [ ! -f "$vkey_file" ]; then
        continue
    fi
    
    # Get the address for this key
    key_addr=$($CARDANO_CLI address build \
        --payment-verification-key-file "$vkey_file" \
        --testnet-magic $TESTNET_MAGIC 2>/dev/null)
    
    if [ -z "$key_addr" ]; then
        continue
    fi
    
    # Query UTXOs for this address
    KEY_UTXOS=$($CARDANO_CLI query utxo \
        --testnet-magic $TESTNET_MAGIC \
        --socket-path "$SOCKET_PATH" \
        --address "$key_addr" 2>/dev/null)
    
    if [ -z "$KEY_UTXOS" ]; then
        continue
    fi
    
    # Check if this address has a UTXO with sufficient funds
    if command -v jq >/dev/null 2>&1; then
        # Parse JSON format
        KEY_UTXO_TOTAL=$(echo "$KEY_UTXOS" | jq -r '[.[] | .value.lovelace] | add // 0' 2>/dev/null)
        KEY_UTXO_TOTAL_INT=$(echo "$KEY_UTXO_TOTAL" | cut -d. -f1)
        
        if [ "$KEY_UTXO_TOTAL_INT" -ge "$REQUIRED_AMOUNT" ]; then
            # Find the first UTXO with sufficient funds
            for utxo_key in $(echo "$KEY_UTXOS" | jq -r 'keys[]' 2>/dev/null); do
                utxo_amount=$(echo "$KEY_UTXOS" | jq -r ".[\"$utxo_key\"].value.lovelace // 0" 2>/dev/null)
                utxo_amount_int=$(echo "$utxo_amount" | cut -d. -f1)
                
                if [ "$utxo_amount_int" -ge "$REQUIRED_AMOUNT" ]; then
                    UTXO_HASH=$(echo "$utxo_key" | cut -d'#' -f1)
                    UTXO_IX=$(echo "$utxo_key" | cut -d'#' -f2)
                    UTXO_AMOUNT="$utxo_amount_int"
                    UTXO_ADDRESS="$key_addr"
                    UTXO_SKEY="$skey_file"
                    UTXO_VKEY="$vkey_file"
                    SOURCE_ADDRESS="$key_addr"
                    break 2
                fi
            done
        fi
    else
        # Fallback: parse text format
        KEY_UTXO_LINE=$(echo "$KEY_UTXOS" | tail -n +3 | head -1)
        if [ -n "$KEY_UTXO_LINE" ]; then
            key_utxo_amount=$(echo "$KEY_UTXO_LINE" | awk '{print $3}')
            if [ "$key_utxo_amount" -ge "$REQUIRED_AMOUNT" ]; then
                UTXO_HASH=$(echo "$KEY_UTXO_LINE" | awk '{print $1}' | cut -d'#' -f1)
                UTXO_IX=$(echo "$KEY_UTXO_LINE" | awk '{print $1}' | cut -d'#' -f2)
                UTXO_AMOUNT="$key_utxo_amount"
                UTXO_ADDRESS="$key_addr"
                UTXO_SKEY="$skey_file"
                UTXO_VKEY="$vkey_file"
                SOURCE_ADDRESS="$key_addr"
                break
            fi
        fi
    fi
done

if [ -z "$UTXO_SKEY" ]; then
    echo_error "Could not find a UTXO with sufficient funds that we can sign"
    echo_info "Searched UTXO keys in: $TESTNET_ENV_DIR/utxo-keys/"
    echo_info "Required: $REQUIRED_AMOUNT lovelace"
    exit 1
fi

echo_success "Found UTXO: ${UTXO_HASH}#${UTXO_IX}"
echo_info "  Amount: $UTXO_AMOUNT lovelace"
echo_info "  Source address: $SOURCE_ADDRESS"
echo_info "  Signing key: $UTXO_SKEY"
echo ""

# Step 2: Calculate change and check minimum UTxO
ESTIMATED_FEE=200000
CHANGE=$((UTXO_AMOUNT - AMOUNT_LOVELACE - ESTIMATED_FEE))

if [ "$CHANGE" -lt 0 ]; then
    echo_error "Insufficient funds in UTXO"
    echo_info "  UTXO amount: $UTXO_AMOUNT lovelace"
    echo_info "  Required: $REQUIRED_AMOUNT lovelace (including fees)"
    exit 1
fi

# Get minimum UTxO requirement (approximately 850000 lovelace for Conway era)
# If change is below minimum, send it all to destination (dust collection)
MIN_UTXO=850000
SEND_CHANGE_TO_DEST=false

if [ "$CHANGE" -lt "$MIN_UTXO" ] && [ "$CHANGE" -gt 0 ]; then
    echo_warn "Change ($CHANGE lovelace) is below minimum UTxO threshold (~$MIN_UTXO lovelace)"
    echo_info "Sending all remaining funds to destination (dust collection)"
    AMOUNT_LOVELACE=$((AMOUNT_LOVELACE + CHANGE))
    CHANGE=0
    SEND_CHANGE_TO_DEST=true
fi

echo_info "Transaction details:"
echo_info "  From: ${UTXO_HASH}#${UTXO_IX}"
echo_info "  To: $WALLET_ADDRESS"
echo_info "  Amount: $AMOUNT_LOVELACE lovelace ($(echo "scale=6; $AMOUNT_LOVELACE / 1000000" | bc) ADA)"
if [ "$CHANGE" -gt 0 ]; then
    echo_info "  Change: $CHANGE lovelace (to $SOURCE_ADDRESS)"
else
    echo_info "  Change: 0 lovelace (all funds sent to destination)"
fi
echo ""

# Step 3: Build transaction
echo_info "Building transaction..."
TX_BODY_FILE=$(mktemp)
TX_SIGNED_FILE=$(mktemp)

# Note: conway transaction build automatically fetches protocol parameters from the node
# Build transaction - if change is too small, send everything to destination
# When using --change-address, cardano-cli will try to create a change output
# If we want to avoid change output when it's too small, we need to calculate the exact amount
if [ "$CHANGE" -gt 0 ] && [ "$CHANGE" -ge 850000 ]; then
    # Normal case: change is above minimum UTxO, create change output
    $CARDANO_CLI conway transaction build \
        --testnet-magic $TESTNET_MAGIC \
        --socket-path "$SOCKET_PATH" \
        --tx-in "${UTXO_HASH}#${UTXO_IX}" \
        --tx-out "${WALLET_ADDRESS}+${AMOUNT_LOVELACE}" \
        --change-address "$SOURCE_ADDRESS" \
        --out-file "$TX_BODY_FILE" 2>&1
else
    # Change is too small or zero - send everything to destination
    # Calculate the actual amount we can send (UTXO amount minus estimated fee)
    ACTUAL_AMOUNT=$((UTXO_AMOUNT - ESTIMATED_FEE))
    $CARDANO_CLI conway transaction build \
        --testnet-magic $TESTNET_MAGIC \
        --socket-path "$SOCKET_PATH" \
        --tx-in "${UTXO_HASH}#${UTXO_IX}" \
        --tx-out "${WALLET_ADDRESS}+${ACTUAL_AMOUNT}" \
        --change-address "$WALLET_ADDRESS" \
        --out-file "$TX_BODY_FILE" 2>&1
fi

if [ $? -ne 0 ]; then
    echo_error "Failed to build transaction"
    cat "$TX_BODY_FILE" >&2
    rm -f "$TX_BODY_FILE" "$TX_SIGNED_FILE"
    exit 1
fi

echo_success "Transaction built"

# Step 4: Sign transaction
echo_info "Signing transaction..."
$CARDANO_CLI conway transaction sign \
    --testnet-magic $TESTNET_MAGIC \
    --tx-body-file "$TX_BODY_FILE" \
    --signing-key-file "$UTXO_SKEY" \
    --out-file "$TX_SIGNED_FILE" 2>&1

if [ $? -ne 0 ]; then
    echo_error "Failed to sign transaction"
    rm -f "$TX_BODY_FILE" "$TX_SIGNED_FILE"
    exit 1
fi

echo_success "Transaction signed"

# Step 5: Submit transaction
echo_info "Submitting transaction..."
TX_HASH=$($CARDANO_CLI conway transaction submit \
    --testnet-magic $TESTNET_MAGIC \
    --socket-path "$SOCKET_PATH" \
    --tx-file "$TX_SIGNED_FILE" 2>&1)

if [ $? -ne 0 ]; then
    echo_error "Failed to submit transaction"
    echo "$TX_HASH" >&2
    rm -f "$PROTOCOL_PARAMS_FILE" "$TX_BODY_FILE" "$TX_SIGNED_FILE"
    exit 1
fi

# Extract transaction hash if it's in the output
if echo "$TX_HASH" | grep -q "Transaction successfully submitted"; then
    TX_HASH=$(echo "$TX_HASH" | grep -oE "[a-f0-9]{64}" | head -1)
fi

echo_success "Transaction submitted!"
echo_info "  Transaction hash: $TX_HASH"
echo ""

# Step 6: Wait for confirmation and verify
echo_info "Waiting for transaction confirmation..."
sleep 3

MAX_RETRIES=10
RETRY_COUNT=0
WALLET_BALANCE=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    WALLET_UTXOS=$($CARDANO_CLI query utxo \
        --testnet-magic $TESTNET_MAGIC \
        --socket-path "$SOCKET_PATH" \
        --address "$WALLET_ADDRESS" 2>/dev/null)
    
    if [ -n "$WALLET_UTXOS" ]; then
        # Calculate balance
        if command -v jq >/dev/null 2>&1; then
            WALLET_BALANCE=$(echo "$WALLET_UTXOS" | jq -r '[.[] | .value.lovelace] | add // 0' 2>/dev/null)
        else
            WALLET_BALANCE=$(echo "$WALLET_UTXOS" | tail -n +3 | awk '{sum+=$3} END {print sum}')
        fi
        
        if [ "$WALLET_BALANCE" -ge "$AMOUNT_LOVELACE" ]; then
            break
        fi
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        sleep 2
    fi
done

# Cleanup
rm -f "$TX_BODY_FILE" "$TX_SIGNED_FILE"

if [ "$WALLET_BALANCE" -ge "$AMOUNT_LOVELACE" ]; then
    WALLET_BALANCE_ADA=$(echo "scale=6; $WALLET_BALANCE / 1000000" | bc)
    echo_success "Wallet funded successfully!"
    echo_info "  Wallet address: $WALLET_ADDRESS"
    echo_info "  Balance: $WALLET_BALANCE lovelace ($WALLET_BALANCE_ADA ADA)"
    echo_info "  Transaction: $TX_HASH"
else
    echo_warn "Transaction submitted but balance not yet confirmed"
    echo_info "  Transaction hash: $TX_HASH"
    echo_info "  Check balance later with:"
    echo_info "    $CARDANO_CLI query utxo --testnet-magic $TESTNET_MAGIC --socket-path $SOCKET_PATH --address $WALLET_ADDRESS"
fi

