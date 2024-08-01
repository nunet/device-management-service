# Introduction

This repository contains implementations for managing contracts, proofs, and payments in tokenomics. Initiated within milestone [], it offers a comprehensive set of interfaces and methods. To implement these functions, we first define key datatypes and interfaces.

## Interfaces and Types

### Contract

The Contract manages the lifecycle of contracts between nodes, including the initiation, signing, notarization, and settlement of contracts. It integrates with other components such as DMS, orchestrators, and payments. It has the following features:

## Features:

- **Contract Management**: Create, sign, notarize, and settle contracts between nodes.
- **Payment Handling**: Manage payment details within contracts.
- **Job Verification**: Integrate job verification results into contract settlement.
- **Database Integration**: Save and update contracts in the central database.

### Proof

The Proof interface defines the methods for proof-based operations. This package includes methods for authentication, contract proof creation, and verification. It has two main components: Authentication and Proof Interface.

### **Authentication**

The Authentication struct offers methods for various authentication mechanisms to ensure secure access control and data validation within the application:

- **tokenBasedEncryption**: Validates credentials using token-based encryption.
- **ZKProof**: Implements zero-knowledge proof for secure authentication.
- **OffChainData**: Handles authentication using off-chain data verification.

### **ProofInterface**

The ProofInterface interface defines essential operations for managing contract proofs:

- **InitiateContractApproval()**: Initiates the contract approval process, starting necessary workflows.
- **CreateContractProof()**: Generates a cryptographic proof for a contract, ensuring transaction integrity.
- **SaveProof(contractID, proof string) error**: Stores the contract proof in a simulated database, maintaining audit trails and historical records.
- **VerifyProof(contractID, proof string) (bool, error)**: Verifies the authenticity of a contract proof, ensuring its validity before further processing.

**Payment**

The Payment interface defines the operations for managing payments and settlements related to contracts. It facilitates payment and pricing functionalities within a blockchain-based payment gateway system. The details of this interface component is given below:

**PaymentChannel**: Defines the types of supported payment methods:

- FiatPayment: Transactions in traditional currencies (e.g., USD, EUR).
- BlockchainPayment: Transactions using blockchain-based cryptocurrencies.

**PricingMethod**: Interface supporting different pricing models:

- FixedJobPricing: Specifies fixed costs for specific tasks, including base price and platform fees.
- PeriodicPricing: Defines recurring pricing details such as price per cycle, duration, usage limits (e.g., CPU hours, memory, storage, bandwidth), and platform fees.

**UsageLimits**: Constraints for periodic pricing models:

- Specifies MaxCPUHours, MaxMemoryUsage, MaxStorageUsage, and MaxNetworkBandwidth allowable within each billing period.

**Functions**

1. **initiateContractClosure():**

**Inputs**:

- **n1**: Node ID (dms.nodeID) representing the first node.
- **n2**: Node ID (dms.nodeID) representing the second node.
- **bid**: Object (orchestrator.Bid) containing job and pricing details for the bestBidSelected.

**Functionality**:

The initiateContractClosure function initializes and closes a contract between two nodes within the system. It follows the sequence:

1. Creates a new contract instance.
2. Populates the contract with job ID and payment details extracted from the provided bid.
3. Signs and notarizes the contract.
4. Persists the contract in the contract lists of both nodes (n1 and n2) and the central database.
5. **initiateContractSettlement():**

**Inputs**:

- **n1**: Node ID (dms.nodeID) representing the first node.
- **n2**: Node ID (dms.nodeID) representing the second node.
- **contract**: Contract ID representing the contract to be settled.
- **verificationResult**: Object (orchestrator.JobVerificationResult) containing the result of job verification.

**Functionality**:

The initiateContractSettlement function initiates the settlement process for a specified contract between two nodes (n1 and n2). It executes the following steps:

1. Updates the contract with the provided verification result.
2. Executes settlement procedures.
3. Marks the contract as settled.
4. Notifies both nodes (n1 and n2) about the settlement.
5. Updates the contract details in the central database.
6. **Authentication()**

**Description**: Contains authentication methods and data related to different authentication techniques given below:

- encryption: Type of encryption used for authentication.
- ZKProof: Details related to Zero-Knowledge Proof authentication.
- OffChain: Data related to off-chain authentication.

### More techniques can be added

**Parameters**:

- nodeID: A variable representing the node being authenticated (dms.node).
- method: The authentication method ("tokenBasedEncryption", "ZKProof", "OffChainData").
- credentials: The credentials required for authentication.

**Functionality:**

The Authenticate method checks the authentication method specified (tokenBasedEncryption, ZKProof, OffChainData) and calls the corresponding authentication function (tokenBasedEncryptionAuthentication, zkProofAuthentication, offChainDataAuthentication). It returns true if authentication is successful based on the provided credentials and false otherwise.

1. **Deposit():**

**Purpose**: The Deposit function manages the deposit logic for payments, distinguishing between direct and escrow methods. It ensures that only valid payment types (fiat or crypto) are accepted for escrow payments. This function is crucial for initiating the payment process based on the specified method and type.

**Parameters**:

- contractID (int): Identifier of the contract associated with the payment.
- payment (Payment): Struct containing details of the payment, including its method (direct or escrow) and payment type (fiat or crypto).
- pricing (PricingMethod): Defines the method used to determine the pricing for the deposit (not fully implemented in this function).

**Returns**: Returns an error if there are issues with the payment method or type, ensuring that only valid configurations proceed with deposit processing.

1. **SettleContract ():**

**Purpose**: The SettleContract function manages the settlement process for contracts based on job verification results. It calculates the payment amount based on the job's completion percentage and processes payments either directly or via escrow, depending on the contract's payment method (direct or escrow). It also handles scenarios where job verification fails and ensures appropriate actions such as refunds for escrow payments.

**Parameters**:

- contractID (int): Identifier of the contract to be settled.
- verificationResult (jobs.JobVerificationResult): Contains the result of job verification, including success status (verificationResult.Success) and completion percentage (verificationResult.Percentage).

**Returns**: Returns an error if there are issues processing payments or refunds, ensuring that settlement operations are executed accurately based on job outcomes.