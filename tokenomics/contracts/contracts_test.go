// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package contracts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalContract(t *testing.T) {
	const data = `
{
    "solution_enabler_did": {
        "uri": "did:key:z6Mkv4TSVeeGP37Sv3vGZWnyVsQK9fbGaButGV6T6eimtHk8"
    },
    "payment_validator_did": {
        "uri": "did:key:z6MkqhCgFzFxUduPMW93jbyvSxHd24QZ5TVkiQpuCCo34yYg"
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
        "requester_addr": "0xe66b31678d6c16e9ebf358268a790b763c133750",
        "provider_addr": "0x4741783ed607d1496f65749d2d9c94cf6c23352a",
        "currency": "NTX",
        "fees_per_allocation": "10",
        "timestamp": "0001-01-01T00:00:00Z",
        "payment_type": "blockchain",
        "blockchain": "ETHEREUM"
    },
    "contract_terms": "Standard contract terms",
    "contract_participants": {
        "provider": {
            "uri": "did:key:z6MknTzQbQHvm8MvrXKYfdphgtsTmpEq7pzF35wHEvY7cSui"
        },
        "requestor": {
            "uri": "did:key:z6MkinfJcDWucYz6uF4rJcYrztX3mXcJt1XZT1cBY6cfnwSY"
        }
    },
    "duration": {
        "start_date": "2024-07-12T18:14:06.993552133+03:00",
        "end_date": "2028-07-12T18:14:06.99355218+03:00"
    }
}
`

	var req CreateContractRequestBehaviour
	err := json.Unmarshal([]byte(data), &req)
	assert.NoError(t, err)
}
