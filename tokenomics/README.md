# tokenomics

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
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

- [README](https://gitlab.com/nunet/device-management-service/-/blob/main/tokenomics/README.md): Current file which is aimed towards developers who wish to use and modify the package functionality.

- [Contract.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/Contract.go?ref_type=heads) Defines the main interface for managing and executing contracts within the tokenomics system.

- [Proofs.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/Proofs.go?ref_type=heads) Implements the interface and logic for proof handling within the tokenomics framework.

- [payments.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/payments.go?ref_type=heads) Contains the main interface and functions for processing payments in the tokenomics system.

- [tokenomics.go:](https://gitlab.com/nunet/open-api/platform-data-model/-/blob/tokenomics-proposed/device-management-service/tokenomics/tokenomics.go?ref_type=heads): Defines the core functionalities and main interface for the tokenomics package, integrating contracts, proofs, and payments.

*Subpackages*

- [./specs/](https://gitlab.com/nunet/device-management-service/-/tree/main/tokenomics/specs): Directory containing package specifications, including package class diagram.
- [./private_ledger/](https://gitlab.com/nunet/device-management-service/tokenomics/private_ledger): Directory containing implementation of private ledger.

- [./Sequences/:](https://gitlab.com/nunet/open-api/platform-data-model/-/tree/proposed/device-management-service/tokenomics/sequences?ref_type=heads) Contains the sequence diagram for the tokenomics package

### 3. Class Diagram

#### Source File

[tokenomics Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/tokenomics/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/tokenomics"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

Note: the functionality of Tokenomics is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested design of interfaces and methods.

### 5. Data Types

Note: the functionality of Tokenomics is being currently developed. See the [proposed](#7-proposed-functionality--requirements) section for the suggested data types.

### 6. Testing

#### Unit Tests

`TBD`

#### Functional Tests

To be determined (`TBD`).

### 7. Proposed Functionality / Requirements

List of issues related to the design of the tokenomics package can be found below. These include proposals for modifications to the package or new functionality needed to cover the requirements of other packages.

- [Tokenomics Package Issues](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=opened&search=tokenomics&first_page_size=20)

### Interfaces and Methods:

#### Proposed Contract Interface

```go

```go
// Contract defines the methods for contract operations
type contract interface {
	NewContract() Contract
	InitiateContractClosure(n1 dms.NodeID, n2 dms.NodeID, bid orchestrator.Bid)
	InitiateContractSettlement(n1 dms.NodeID, n2 dms.NodeID, contractID int, verificationResult orchestrator.JobVerificationResult)
	processContractSettlement(ctx context.Context, contract *Contract, verificationResult jobs.Status) 
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
2. Handles settlement based on the job status and processes payments.
3. Notifies both nodes (n1 and n2) about the settlement.
4. Updates the contract details in the central database.

**ProcessContractSettlement:** processes the contract settlement based on the pricing method and verification result
1. Calculates payment based on the pricing method and processes it.
2. Handles job failure by issuing refunds if required.

#### Proposed Proof Interface

```go

ProofInterface defines the methods for handling proof-based operations
type proofs interface {
	InitiateContractApproval() 
	CreateContractProof() 
	SaveProof()
	VerifyProof()
}
```

**The InitiateContractApproval():** initiates the contract approval process, starting necessary workflows.

**The CreateContractProof():** generates a cryptographic proof for a contract, ensuring transaction integrity.

**The SaveProof(contractID, proof string) error:**  stores the contract proof in a simulated database, maintaining audit trails and historical records.

**The VerifyProof(contractID, proof string) (bool, error):** verifies the authenticity of a contract proof, ensuring its validity before further processing.

#### **Proposed Payment Interface**

```go
// Payment defines the operations for managing payments and settlements
type PaymentGateway interface {
	Deposit(contractID int, payment Payment) error
	SettleContract(contractID int, verificationResult jobs.JobVerificationResult) error
}
```

**Deposit:**  manages the deposit logic for payments, distinguishing between direct and escrow methods. It ensures that only valid payment types (fiat or crypto) are accepted for escrow payments. This function is crucial for initiating the payment process based on the specified method and type.

**Parameters**:

- contractID (int): Identifier of the contract associated with the payment.
- payment (Payment): Struct containing details of the payment, including its method (direct or escrow) and payment type (fiat or crypto).


**SettleContract:**  manages the settlement process for contracts based on job verification results. It calculates the payment amount based on the job's completion percentage and processes payments either directly or via escrow, depending on the contract's payment method (direct or escrow). It also handles scenarios where job verification fails and ensures appropriate actions such as refunds for escrow payments.

  ```go

  // PricingMethod is an interface that can hold either FixedJobPricing or PeriodicPricing
  type PricingMethod interface {
    GetFixedJobPricing() *FixedJobPricing
    GetPeriodicPricing() *PeriodicPricing
}
```


**FixedJobPricing:** It holds pricing details for jobs with fixed payment terms.

**PeriodicPricing:** It is used for jobs with periodic payment structures. It also includes usage limits to define quotas.


### **Data types** `proposed`

**proposed tokenomics.Contract:** Consists of detailed information regarding an agreement between a requestor and a provider within the network. This data type includes the following fields:


```go
// Contract struct defines a contract with the following fields:
type Contract struct {
	ContractID     int
	JobID          int         // Example default: 1001
	PaymentDetails Payment     // Zero value: zero value of payments.Payment struct
	Signatures     []byte      // Changed to []byte to hold binary signature data or encrypted
	Settled        bool        // Example default: false
	Verification   jobs.Status // Zero value: zero value of jobs.Status
	ContractProof  string      // Example default: "Pending"
}
```

**tokenomics.Payment**: Consists of details related to a payment transaction between a requestor and a provider, specifying the type, channel, currency, pricing method, and timestamp of the transaction.

---

```go

// Payment represents a payment transaction
type Payment struct {
    Requestor      string
    Provider       string
	Currency       string 
    Timestamp      time.Time
    PaymentType    string          //PaymentType (like escrow vs. direct) 
    PaymentChannel PaymentChannel 
	Pricing        PricingMethod
   
}

type PricingMethod interface {
    GetFixedJobPricing() *FixedJobPricing
    GetPeriodicPricing() *PeriodicPricing
}
```

**tokenomics.FixedJobPricing:** Consists of information related to the fixed pricing for a job, detailing the cost and platform fee involved.

```go

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

### Private_ledger
The `private_ledger` sub package provides a `DatabaseManager` to manage PostgreSQL databases for contracts database. It allows users to initialize database connections, insert contract data, retrieve contract, and close connections safely.

### Features

- **Database Initialization**: Create and manage a connection to a PostgreSQL database.
- **Contract Retrieval**: Fetch all records from a specified  table.
- **Storing Contract**: Insert contract records into table.
- **Connection Management**: Close database connections safely.

### 8. References

- [proposed design](https://www.notion.so/Tokenomics-2e3696cde66a4179b96e9a3a9daeaa10?pvs=21)

