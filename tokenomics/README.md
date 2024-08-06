# tokenomics

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/develop/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/develop/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution guidelines](https://gitlab.com/nunet/device-management-service/-/blob/develop/CONTRIBUTING.md)
- [Code of conduct](https://gitlab.com/nunet/device-management-service/-/blob/develop/CODE_OF_CONDUCT.md)
- [Secure coding guidelines](https://gitlab.com/nunet/documentation/-/wikis/secure-coding-guidelines)

## Table of Contents

1. [Description](#1-description)
2. [Structure and organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

This repository contains implementations for managing contracts, proofs, and payments in tokenomics. Initiated within milestone [Device Management Service Version 0.5.x](https://gitlab.com/groups/nunet/-/milestones/44#tab-issues), it offers a comprehensive set of interfaces and methods. To implement these functions, we first define key datatypes and interfaces.

### 2. Structure and Organisation

Here is quick overview of the contents of this directory:

- [README](https://gitlab.com/nunet/device-management-service/-/blob/develop/tokenomics/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

- [Contract.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/Contract.go?ref_type=heads) Defines the main interface for managing and executing contracts within the tokenomics system.

- [Proofs.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/Proofs.go?ref_type=heads) Implements the interface and logic for proof handling within the tokenomics framework.

- [payments.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/payments.go?ref_type=heads) Contains the main interface and functions for processing payments in the tokenomics system.

- [tokenomics.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/tokenomics.go?ref_type=heads): Defines the core functionalities and main interface for the tokenomics package, integrating contracts, proofs, and payments.

*Subpackages*

- [./specs/](https://gitlab.com/nunet/device-management-service/-/tree/develop/tokenomics/specs): Directory containing package specifications, including package class diagram.

- [./Sequences/:](https://gitlab.com/nunet/open-api/platform-data-model/-/tree/proposed/device-management-service/tokenomics/sequences?ref_type=heads) Contains the sequence diagram for the tokenomics package

### 3. Class Diagram

#### Source File

[tokenomics Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/develop/tokenomics/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/develop"
!$packageRelativePath = "/tokenomics"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

Note: the functionality of Tokenomics is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested design of interfaces and methods.

### 5. Data Types

Note: the functionality of DMS is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested data types.

### 6. Testing

#### Unit Tests

`TBD`

#### Functional Tests

To be determined (`TBD`).

### 7. Proposed Functionality / Requirements

List of issues related to the design of the tokenomics package can be found below. These include proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [Tokenomics Package Issues](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&search=tokenomics&first_page_size=20)

#### Interfaces and Methods:

##### Proposed Contract Interface

```go
// Contract defines the methods for contract operations
type contract interface {
	NewContract() Contract
	InitiateContractClosure(n1 dms.NodeID, n2 dms.NodeID, bid dms.orchestrator.Bid)
	InitiateContractSettlement(n1 dms.NodeID, n2 dms.NodeID, contractID int, verificationResult dms.orchestrator.JobVerificationResult)
}
```

**NewContract()**: Creates new contract
**InitiateContractClosure:** function initializes and closes a contract between two nodes within the system. It follows the sequence:

1. Creates a new contract instance.
2. Populates the contract with job ID and payment details extracted from the provided bid.
3. Signs and notarizes the contract.
4. Persists the contract in the contract lists of both nodes (n1 and n2) and the central database.

 **InitiateContractSettlement:** function initiates the settlement process for a specified contract between two nodes (n1 and n2). It executes the following steps:

1. Updates the contract with the provided verification result.
2. Executes settlement procedures.
3. Marks the contract as settled.
4. Notifies both nodes (n1 and n2) about the settlement.
5. Updates the contract details in the central database.

##### Proposed Proof Interface

```go

type proofs interface {
	InitiateContractApproval() error
	CreateContractProof() (string, error)
	SaveProof(contractID, proof string) error
	VerifyProof(contractID, proof string) (bool, error)
}
```

**The InitiateContractApproval():** initiates the contract approval process, starting necessary workflows.

**The CreateContractProof():** generates a cryptographic proof for a contract, ensuring transaction integrity.

**The SaveProof(contractID, proof string) error:**  stores the contract proof in a simulated database, maintaining audit trails and historical records.

**The VerifyProof(contractID, proof string) (bool, error):** verifies the authenticity of a contract proof, ensuring its validity before further processing.

##### **Proposed Payment Interface**

```go
// Payment defines the operations for managing payments and settlements
type payments interface {
	Deposit(contractID int, payment tokenomics.Payment, pricing tokenomics.PricingMethod) error
	SettleContract(contractID int, verificationResult dms.jobs.JobVerificationResult) error
}
```

**Deposit:**  manages the deposit logic for payments, distinguishing between direct and escrow methods. It ensures that only valid payment types (fiat or crypto) are accepted for escrow payments. This function is crucial for initiating the payment process based on the specified method and type.

**Parameters**:

- contractID (int): Identifier of the contract associated with the payment.
- payment (Payment): Struct containing details of the payment, including its method (direct or escrow) and payment type (fiat or crypto).
- pricing (PricingMethod): Defines the method used to determine the pricing for the deposit (not fully implemented in this function).

  **SettleContract:**  manages the settlement process for contracts based on job verification results. It calculates the payment amount based on the job's completion percentage and processes payments either directly or via escrow, depending on the contract's payment method (direct or escrow). It also handles scenarios where job verification fails and ensures appropriate actions such as refunds for escrow payments.

##### **Data types** `proposed`

**proposed tokenomics.Contract:** Consists of detailed information regarding an agreement between a requestor and a provider within the network. This data type includes the following fields:

```go
// Contract represents the contract details between nodes
type Contract struct {
	ContractID     int //A unique identifier for the contract.
	JobID          int  //The identifier of the job associated with the contract.
	Requestor      string  //The entity requesting the service.
	Provider       string  //  The entity providing the service.
	PaymentDetails tokenomics.Payment  //An instance of the payments.Payment type, detailing the payment arrangements for the contract.
	Signatures     []dms.nodeID     //A slice of dms.nodeID values, representing the digital signatures of involved parties.
	Settled        bool       //A boolean indicating whether the contract has been settled.
	Verification   dms.orchestrator.JobVerificationResult  //An instance of the orchestrator.JobVerificationResult type, containing the result of the job verification process.
	ContractProof  dms.orchestrator.ContractProof    // An instance of the orchestrator.ContractProof type, providing proof of the contract's terms and conditions.

}
```

**tokenomics.Payment**: Consists of details related to a payment transaction between a requestor and a provider, specifying the type, channel, currency, pricing method, and timestamp of the transaction.

---

```go

type Payment struct {
    Requestor      string        // The entity initiating the payment
    Provider       string        // The entity receiving the payment
    Currency       string        // The currency in which the payment is made
    Timestamp      time.Time     // The time when the payment was made
    PaymentType    string        // The type of payment (e.g., escrow, direct)
    PaymentChannel PaymentChannel // The channel through which the payment is processed
    Pricing        PricingMethod  // The method used for pricing the payment
}

type PricingMethod struct {
    `TBD`
}

type PaymentChannel struct {
    `TBD`
}
```

**tokenomics.FixedJobPricing:** Consists of information related to the fixed pricing for a job, detailing the cost and platform fee involved.

```go
goCopy code
// FixedJobPricing represents the details for fixed job pricing
type FixedJobPricing struct {
    // Price is the total cost for the fixed job.
    Price int
    // PlatformFee is the fee charged by the platform for the fixed job.
    PlatformFee int
}
```

---

**tokenomics.PeriodicPricing:** Consists of information related to the periodic pricing model, including the cost, period, usage limits, and platform fee.

```go
goCopy code
// PeriodicPricing represents the details for periodic pricing
type PeriodicPricing struct {
    // Price is the cost for the periodic service.
    Price int
    // Period is the duration of the pricing period (e.g., monthly, yearly).
    Period string
    // UsageLimits defines the maximum allowed usage for resources within the pricing period.
    UsageLimits tokenomics.UsageLimits
    // PlatformFee is the fee charged by the platform for the periodic service.
    PlatformFee int
}
```

---

**tokenomics.UsageLimits:** Consists of information regarding the resource usage limits or quotas associated with periodic pricing, specifying the maximum allowable usage for various resources.

```go
goCopy code
// UsageLimits represents the usage limits or quotas for periodic pricing
type UsageLimits struct {
    // MaxCPUHours is the maximum number of CPU hours allowed within the pricing period.
    MaxCPUHours int
    // MaxMemoryUsage is the maximum amount of memory usage allowed within the pricing period.
    MaxMemoryUsage int
    // MaxStorageUsage is the maximum amount of storage usage allowed within the pricing period.
    MaxStorageUsage int
    // MaxNetworkBandwidth is the maximum network bandwidth usage allowed within the pricing period.
    MaxNetworkBandwidth int
}
```

---

**tokenomics.Authentication:** type is designed to handle the authentication details necessary for secure transaction processing within the payment gateway system. This type includes:

- **Encryption**: Specifies the encryption method or protocol used to protect the data involved in the authentication process, ensuring that data is transmitted securely and is kept confidential from unauthorized parties.
- **ZKProof**: Contains the zero-knowledge proof (ZKProof) which allows the verification of the transaction's authenticity without exposing sensitive information. This proof ensures that the transaction is valid while preserving privacy.
- **OffChain**: Represents off-chain data that supports the authentication process. This data includes information not stored directly on the blockchain but is essential for validating and processing transactions securely.

```go
type Authentication struct {
    // encryption: Defines the encryption protocol used to protect data integrity and confidentiality during the authentication process.
    encryption string

    // ZKProof: Contains the zero-knowledge proof that allows verification of the authentication without disclosing sensitive information.
    ZKProof string

    // OffChain: Holds off-chain data that is essential for the authentication process but not stored on the blockchain.
    OffChain OffChainData
}

type OffChainData struct {
    `TBD`
}
```


### 8. References

- [proposed design](https://www.notion.so/Tokenomics-2e3696cde66a4179b96e9a3a9daeaa10?pvs=21)

