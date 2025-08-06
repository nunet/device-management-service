# Testing contracts functionality

### External agreement

We need to setup capabilities between the participating parties. That is to allow `/tokenomics` namespace capabilities on the participants.

Once we have this ready, we need to create a contract file and place the contract details and the participants dids:

```
{
    "solution_enabler_did": {
        "uri": "{{solutionEnablerDID}}"
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
        "Requestor": "",
        "Provider": "",
        "Currency": "",
        "Timestamp": "0001-01-01T00:00:00Z",
        "PaymentType": "direct",
        "PaymentMode": "one-time",
        "PricingMeta": {
            "Price": 100,
            "PlatformFee": 10,
            "Type": "fixed",
            "Periodic": null
        }
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
}
```

### Create the contract

Requester has the above contract file saved, and filled. Now we need to send a contract creation request to the contract host.

From the requester machine we execute the following command:

```
dms actor cmd --context yourcontexthere /dms/tokenomics/contract/create --contract-file path_to_contract_file --timeout 5s --dest did_of_contract_host
```

The following command returns the contracts DID and will be shown in the output

After this, the contract host will propose to the provider the contract which has to be signed by the provider.

On the provider side, we can list the incoming contracts awaiting approval:

```
dms actor cmd --context yourcontexthere /dms/tokenomics/contract/list_incoming --timeout 5s
```

#### Approving a contract from provider side

From the above list, we can see the contract DID. we can approve the desired contract by executing:

```
dms actor cmd --context yourcontexthere /dms/tokenomics/contract/approve_local --contract-did contract_did_from_above_list --timeout 5s
```

This will sign and aknowledgement to the contract host.

At this stage a contract is accepted


#### Checking contract state on contract host

```
dms actor cmd --context yourcontexthere /dms/tokenomics/contract/state --contract-did contract_did_from_above_list --timeout 5s --dest "{contract_actor_handle}"
```

This will return the contract itself and we can inspect its status

#### Other behaviours

Similar to the above behavior we can run other behaviours including:

```
"/dms/tokenomics/contract/terminate"
"/dms/tokenomics/contract/complete"
"/dms/tokenomics/contract/state"
"/dms/tokenomics/contract/settle"
"/dms/tokenomics/contract/validate"
```

### Deploy with a contract

In the ensemble we can supply the contract details this way:

```
version: "V1"

contracts:
    contract1:
        did: contract_did
        host: contract_host_did


allocations:
    helloAlloc:
        type: task
        executor: docker
        resources:
            cpu:
                cores: 1
            gpus: []
            ram:
                size: 1
            disk:
                size: 1
        execution:
            type: docker
            image: hello-world
            working_directory: /
        keys: []
        provision: []
        dns_name: hellodocker

nodes:
    node1:
        allocations:
            - helloAlloc
```
