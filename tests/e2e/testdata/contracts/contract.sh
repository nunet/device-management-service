#!/bin/bash

# Check if required arguments are provided
if [ "$#" -lt 4 ]; then
  echo "Usage: $0 <solutionEnablerDID> <paymentValidatorDID> <providerDID> <requesterDID>"
  exit 1
fi

# Assign arguments to variables
solutionEnablerDID="$1"
paymentValidatorDID="$2"
providerDID="$3"
requesterDID="$4"

# You can still hardcode or extend more vars if needed
requesterAddr="0x82Ef4B08436E97F991Dd4CF71d998C3750514bE8"
providerAddr="0x990b842329C82ab1AF97579005060c23B7a3E650"
amount="500"

# JSON template
template='{
    "solution_enabler_did": {
        "uri": "{{solutionEnablerDID}}"
    },
    "payment_validator_did": {
        "uri": "{{paymentValidatorDID}}"
    },
    "resource_configuration": {
        "cpu": {
            "clock_speed": 0,
            "cores": 1
        },
        "ram": {
            "size": 1024
        },
        "disk": {
            "size": 1024
        }
    },
    "termination_option": {
        "allowed": true,
        "notice_period": 86400000000000
    },
    "penalties": [
        {
            "condition": "late",
            "penalty": 100
        }
    ],
    "payment_details": {
        "addresses": [
            {
                "requester_addr": "{{requesterAddr}}",
                "provider_addr": "{{providerAddr}}",
                "currency": "NTX",
                "blockchain": "ETHEREUM"
            }
        ],
        "fees_per_allocation": "{{amount}}",
        "timestamp": "0001-01-01T00:00:00Z",
        "payment_type": "blockchain"
    },
    "contract_terms": "Standard contract terms",
    "contract_participants": {
        "provider": {
            "uri": "{{providerDID}}"
        },
        "requestor": {
            "uri": "{{requesterDID}}"
        }
    },
    "duration": {
        "start_date": "2024-07-12T18:14:06.993552133+03:00",
        "end_date": "2028-07-12T18:14:06.99355218+03:00"
    }
}'

# Replace placeholders
filled=$(echo "$template" | \
  sed "s|{{solutionEnablerDID}}|$solutionEnablerDID|g" | \
  sed "s|{{paymentValidatorDID}}|$paymentValidatorDID|g" | \
  sed "s|{{requesterAddr}}|$requesterAddr|g" | \
  sed "s|{{providerAddr}}|$providerAddr|g" | \
  sed "s|{{amount}}|$amount|g" | \
  sed "s|{{providerDID}}|$providerDID|g" | \
  sed "s|{{requesterDID}}|$requesterDID|g")

# Print final JSON
echo "$filled"