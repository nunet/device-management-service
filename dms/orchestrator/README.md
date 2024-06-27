# Introduction
This package takes care of job scheduling and management (manages jobs on other DMSs)

# Job Orchestration

The lifecyle of a job on Nunet platform consists of various operations from job posting to settlement of the contract. A key distinction to note is the option of two types of orchestration mechanisms: `push` and `pull`. Broadly speaking `pull` orchestration works on the premise that resource providers bid for jobs available in the network, while `push` orchestration works when a job is `push`ed directly to a known resource provider -- constituting to a more centralized orchestration. `push` orchestration develops on the idea that users choose from the available providers and their resources. However, given the decentralized and open nature of the platform, it may be required to engage the providers to get their current (latest) state and preferences. This leads to an overlap with the `pull` orchestration approach.

The default setting is to use `pull` based orchestration, which is mostly developed in the present proposed specification. However, the user can choose to use `push` based orchestration to suit their needs.

A design involving both mechanisms allows us to cover continuum between radically decentralized and centralized orchestration styles in the same system. The details of all the operations involved in the job orchestration are described in the following sections.

Job orchestration mechanism spans across many dms packages, including `orchestrator` itself, but also `jobs`, `network`, `executor`, etc. This specification aims at describing the whole process which will be then split into different packages for implementation.

## Related interfaces and types

* `dms.orchestrator.Orchestrator` is the interface combining methods that are needed for orchestration and do not fit into other packages or interfaces;

## Related functions / methods

This is a list of function proposed for implementation in `Orchestrator` interface. The list as well as a description is not complete nor they are completely defined. But these functions are the main to implement orchestrator functionality. It may be, that some of them will move to other packages during consolidation.

* `dms.orchestrator.orchestrateJob` is called whenever a dms receives a `jobPosting`;
* `dms.orchestrator.jobRequest`;
* `dms.orchestrator.jobInvocation`;
* `dms.orchestrator.publishBidRequest`;
* `dms.orchestrator.compare`;
* `dms.orchestrator.acceptJob`
* `dms.orchestrator.bid` -- resource provider function to bid for a requeted job;
* `dms.orchestrator.registerBid`
* `dms.orchestrator.selectBestBid`
* `dms.orchestrator.invokeAllocation` -- invoke an allocation of a job on a machine with which a contract was signed;

## 1. Job Posting

* _proposed 2024-03-21; last updated 2024-05-07; by: @kabir.kbr; @janaina.senna; @0xPravar_

This is based on preliminary design, please refer to [research/blog/jobposting](https://nunet.gitlab.io/research/blog/posts/job-orchestration-details/#1-job-posting).

The first step is when a user posts a request to run a computing job. This should define various job requirements and preferences.

**endpoint**: `/orchestrator/postJob`<br/>
**method**: `HTTP POST`<br/>
**output**: `None`

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Job_Posting.feature))   |
| Request payload       | [jobDescription](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/jobs/data/jobDescription.payload.go)|
| Return payload       | None |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/jobPosting.sequence.mermaid),[.svg](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/rendered/jobPosting.sequence.svg)) | 

### List of relevant functions

* `dms.orchestrator.processJob(JobDescription) -> Job` - This function will validate the job received, add metadata (if needed) and save the job to the local database.
_notes: most probably this functionality will have to move to api/cmd package, as job posting will happen via these interfaces; the processing of the messages received via these interfaces may be done in general RPC-style, as proposed in Node and Allocation interfaces_

### List of relevant data types

_note: related to the dms.jobs package; will need to be moved there;_

* `dms.jobs.Job` -- a type containing all information about a requested job; Note, that `Job` is defined as a recursive structure, allowing any job to have child jobs;

* `dms.jobs.JobLink` -- expresses links between `Jobs` in a parent-child structure;

* `dms.jobs.Pod` --  a collection of jobs that need to be deployed on a single machine and therefore should be provided by amount of resources that are needed for all these jobs;

* `dms.jobs.Allocation` -- an interface and type, encapsulating each job that has been allocated and executed on a specific resource provider;

## 2. Search and Match

* _proposed 2024-03-27; by: @kabir.kbr; @janaina.senna; @0xPravar_

Once the DMS received a job posting, it will look to find nodes that can service the request. This is done by matching the job requirements with the available resources.

### Configuration

`dms.dms.config.defaultOrchestrationType`: The default setting of the network is use `pull` search and match operation. This value is stored in `defaultOrchestrationType` parameter saved in the [config](https://gitlab.com/nunet/device-management-service/-/tree/proposed/dms/config) folder under `dms` package. The user can change this value to `push` if needed via CLI (Command Line Interface) or API. This functionality is covered in the [dms](https://gitlab.com/nunet/device-management-service/-/tree/proposed/dms) folder.

`dms.dms.config.defaultSearchTimeout`: Each DMS will have default timeout value for the search operation. This value is stored in `defaultSearchTimeout` parameter saved in the [config](https://gitlab.com/nunet/device-management-service/-/tree/proposed/dms/config) folder under `dms` package. This can be overridden by the owner of the DMS via CLI (Command Line Interface) or API. This functionality is covered in the [dms](https://gitlab.com/nunet/device-management-service/-/tree/proposed/dms) folder.

### Pull Based

The first step is that service provider DMS requests bids from the compute providers in the network. DMS on compute provider compares the capability of the available resources against soft and hard constraints specified in the job requirements. If all the requirements are met, it then decides whether to submit a bid. The final outcome is that service provider DMS has a list of eligible compute providers with their bids.

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Pull_Search_And_Match.feature))   |
| Request payload       | [BidRequest](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/bidRequest.payload.go)|
| Data at rest (CP DMS)      | [AvailableCapability](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/dms/data/availableCapability.payload.go) |
| Data at rest (CP DMS)      | [CapabilityComparison](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/capabilityComparison.payload.go) |
| Return payload       | [Bid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/bid.payload.go) |
| Data at rest (SP DMS)       | [EligibleComputeProvidersIndex](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/computeProviderIndex.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/pullSearchAndMatch.sequence.mermaid),[.svg]()) |

The second step is to shortlist the preferred compute provider peer based on some selection criteria. Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Select_Preferred_Node.feature))   |
| Request payload       | [EligibleComputeProvidersIndex](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/computeProviderIndex.payload.go) |
| Return payload       | [EligibleComputeProviderData](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/computeProviderIndex.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/selectPreferredNode.sequence.mermaid),[.svg]()) |

#### List of relevant functions

* `dms.orchestrator.publishBidRequest(dms.orchestrator.BidRequest, ...topic)`: the function for publishing the bidRequest (formerly publishJob function) to the network via gossipsub protocol implementation, which allows us to broadcast messages to several nodes (selected by a topic simultaneously); the `topic` parameter for this function is optional. See more detailed proposal of the function in [open-api/platform-data-model/orchestrator/orchestrator.go](https://gitlab.com/nunet/open-api/platform-data-model/-/tree/proposed/device-management-service/orchestrator/orchestrator.go). 

`dms.dms.availableResources()` - This function will return `dms.dms.availableCapability` which is the available resources/capability of the machine to perform a job.

`dms.orchestrator.compare()` - This function takes `dms.jobs.jobDescription.requiredCapability` and `dms.dms.availableCapability` as input and returns `dms.orchestrator.capabilityComparison`.

`dms.orchestrator.accept()` - This function takes `dms.orchestrator.capabilityComparison` and `dms.dms.config.capabilityComparator` as inputs and returns a `bool` which indicates whether the job can be accepted or not.

`dms.orchestrator.createBid()` - This function returns `dms.orchestrator.bid` which is to be sent to service provider DMS.

`dms.orchestrator.registerComputeBid()` - This function takes `dms.orchestrator.bid` as inputs and saves it to a table in the local database as `dms.orchestrator.computeProvidersIndex`.

#### List of relevant data types

`dms.orchestrator.bidRequest` - This is sent to the compute providers based on which they can submit a bid.

`dms.orchestrator.bid` - This is the bid that is submitted by the compute provider.

`dms.dms.availableCapability` - This contains the available resources/capability of the machine to perform a job.

`dms.orchestrator.capabilityComparison` - This contains the comparison of the job requirements with the available resources.

`dms.orchestrator.computeProvidersIndex` - This contains the data of compute providers whose bids have been received. 

## 3. Job Request

* _proposed 2024-03-27; by: @kabir.kbr; @janaina.senna; @0xPravar_

DMS on service provider side checks whether the resources are locked by the preferred compute provider peer.

In case the shortlisted compute provider has not locked the resources while submitting the bid, the job request workflow is executed. This requires the compute provider DMS to lock the necessary resources required for the job and re-submit the bid. Note that at this stage compute provider can still decline the job request.

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Job_Request.feature))   |
| Request payload       | [BidRequest](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/bidRequest.payload.go) |
| Return payload - request acceptance      | [Bid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/bid.payload.go) |
| Return payload - request denied      | [DeclineMessage](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/declineJobRequest.payload.go) |
| Return payload - timeout      | [TimeoutResponse](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/timeoutJobRequest.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/jobRequest.sequence.mermaid),[.svg]()) |

### List of relevant functions

`dms.orchestrator.checkResourceLock()` - This function takes a value from `dms.orchestrator.computeProvidersIndex` as input and checks if the resources required for the job are locked by the compute provider. It returns a `bool` value as output.

`dms.network.queryPeer()` - This function sends the request to lock resources to the the compute provider DMS. It takes `peerID` of the compute provider and `dms.orchestrator.bidRequest` as inputs.

`dms.orchestrator.accept()` - This function decides whether to accept the job request. It takes `dms.orchestrator.bidRequest` as input and returns a `bool` value.

`dms.orchestrator.lockResources()` - This function locks the necessary resources required for the job. It takes `dms.jobs.jobDescription` as input and returns `dms.orchestrator.bid`.

`dms.database.saveEvent()` - This function saves the event to the local database. Input value is TBD.

### List of relevant data types

`dms.orchestrator.computeProvidersIndex` - This contains the data of compute providers whose bids have been received. 

`dms.orchestrator.bidRequest` - This is sent to the chosen compute provider as part of job request.

`dms.orchestrator.bid` - This is the bid that is returned by the compute provider after locking of resources for the job.

`dms.orchestrator.declineJobRequest` - Message sent to the service provider if compute provider declines the job request.

`dms.orchestrator.timeoutJobRequest` - Message sent to the service provider if timeout happens.

`dms.database.declineJobEvent` - TBD


## 4. Contract Closure

* _proposed 2024-03-27; by: @kabir.kbr; @janaina.senna; @0xPravar_

The service provider and the shortlisted compute provider verify that the counterparty is a verified entity and approved by Nunet Solutions to participate in the network. This in an important step to establish trust before any work is performed.

If job does not require any payment (Volunteer Compute), contract is generated by both Service Provider and Compute Provider DMS. This is then verified by `Contract-Database`. 

Otherwise, proof of contract needs to be received from the `Contract-Database` before start of work

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Contract_Closure.feature))   |
| Request payload - payment included      | [ID](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/dms/config/data/id.payload.go) |
| Request payload - payment not included      | [Contract](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/tokenomics/data/contract.payload.go) |
| Return payload - Contract Exists      | [Contract](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/tokenomics/data/contract.payload.go) |
| Return payload - No Contract      | [ErrorMessage](TBD) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/contractClosure.sequence.mermaid),[.svg]()) |

### List of relevant functions
`dms.orchestrator.checkPaymentInfo()` - This function takes `dms.orchestrator.bidRequest` as input and checks if payment is included. It returns a bool value.

`dms.orchestrator.generateJobContract()` - This function takes `dms.orchestrator.bidRequest` as input and creates a contract on Service Provider DMS. It returns `dms.tokenomics.contract`.

`dms.orchestrator.generateJobAgreement()` - This function takes `dms.orchestrator.bidRequest` as input and creates a agreement on Compute Provider DMS. It returns `dms.tokenomics.contract`.

`dms.network.validateProvider()` - This function takes the ID of DMS (`dms.dms.config.ID`) as input and sends it to the `Contract-Database` to check whether this DMS has a valid contract. 

`contract-database.checkKYC()` - This function checks whether the DMS had done KYC and is a verified entity with a valid ID. It takes `dms.dms.config.ID` as input and returns `bool`.

`contract-database.checkContract()` - This functions takes `dms.dms.config.ID` as input and checks if the DMS whose ID has been provided has a valid contract. It returns `dms.tokenomics.contract` as proof of contract or an error message if contract does not exist.

`dms.database.saveEvent()` - This function saves the event to the local database when contract does not exist. Input value is TBD.

### List of relevant data types

`dms.orchestrator.bidRequest` - This is the bid request sent propagated in the network or sent to the chosen compute provider as part of job request.

`dms.tokenomics.contract` - This contains the contract data as well as proof of contract which will mention the entity that provides trust. 

`dms.dms.config.ID` - This contains identifiers like UUID, Peer ID and DID for the DMS.

## 5. Invocation and Allocation

* _proposed 2024-03-29; by: @kabir.kbr; @janaina.senna; @0xPravar_

When the contract closure workflow is completed, both the service provider and compute provider DMS have an agreement and proof of contract with them. Then the service provider DMS will send an invocation to the compute provider DMS which results in job allocation being created. Allocation can be understood as an execution space / environment on actual hardware that enables a Job to be executed.

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Invocation_And_Allocation.feature))   |
| Request payload     | [Invocation](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/invocation.payload.go) |
| Data at rest       | [Allocation](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/executor/data/allocation.payload.go) |
| Return payload      | [AllocationStartSuccess](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/allocationStartSuccess.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/invocationAndAllocation.sequence.mermaid),[.svg]()) |

### List of relevant functions

`dms.network.sendInvocation()` - This function sends the invocation to the compute provider DMS. It takes `dms.orchestrator.invocation` as input.

`dms.executor.createAllocation()` - This function creates an allocation on the compute provider DMS. It takes `dms.orchestrator.invocation` as input and returns `dms.executor.Allocation`.

### List of relevant data types

`dms.orchestrator.invocation` - Invocation which is sent to the compute provider DMS. This contains job description and contract data.

`dms.executor.Allocation` - This contains identifier of the allocation being created along with its status and errors (if any).

`dms.orchestrator.allocationStartSuccess` - This is the response from the compute provider DMS to Service Provider once allocation has been created.

## 6. Job Execution

* _proposed 2024-03-29; by: @kabir.kbr; @janaina.senna; @0xPravar_

Once allocation is created, the job execution starts on the compute provider machine. 

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Job_Execution.feature))   |
| Request payload     | None |
| Return payload - success     | [Result](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/jobs/data/result.payload.go) |
| Return payload - error     | [Result](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/executor/data/error.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/jobExecution.sequence.mermaid),[.svg]()) |

### List of relevant functions

`dms.executor.jobUpdate()` - This function sends job updates to the service provider while execution. It takes `dms.executor.allocation.AllocationID` as input and returns `dms.executor.jobStatusUpdate`.

### List of relevant data types

`dms.executor.jobStatusUpdate` - This is the status update sent by the compute provider DMS to the service provider DMS during job execution.

`dms.jobs.result` - This includes the outcome of the work done by the compute provider along with proof.

`dms.jobs.jobCompleted` - This is sent to Oracle after job is completed.

`dms.executor.error` - This is an error response sent to Service Provider DMS in case of errors during job execution.

## 7. Contract Settlement

* _proposed 2024-03-29; by: @kabir.kbr; @janaina.senna; @0xPravar_

After job is completed, service provider verifies the work done using `Oracle`. If the work is correct, the `Contract-Database` makes the necessary transactions to settle the the contract.

Please see below for relevant specification and data models.

| Spec type              | Location |
---|---|
| Features / test case specifications | Scenarios ([.gherkin](https://gitlab.com/nunet/test-suite/-/blob/proposed/stages/functional_tests/features/device-management-service/orchestrator/Contract_Settlement.feature))   |
| Request payload     | [JobVerification](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/jobVerification.payload.go) |
| Return payload      | [Message](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/data/message.payload.go) |
| Processes / Functions | sequenceDiagram ([.mermaid](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/proposed/device-management-service/orchestrator/sequences/contractSettlement.sequence.mermaid),[.svg]()) |

### List of relevant functions

`oracle.verifyJob()` - This function sends the job and contract data to the Oracle for verification. It takes `dms.orchestrator.jobVerification` as input and returns `dms.orchestrator.jobVerificationResult`.

### List of relevant data types

`dms.orchestrator.jobVerification` - This contains job description and contract data along with job result. This is the data that is processed by Oracle to verify the job.

`dms.orchestrator.jobVerificationResult` - This contains the result of the job verification done by Oracle.

`dms.orchestrator.message` - This is the message that is sent to the other DMS after contract is settled and business ends.

