# tokenomics

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
3. [Implementation Status](#3-implementation-status)

## Specification

### 1. Description

This package contains implementations for managing contracts, proofs, and payments in tokenomics.

### 2. Structure and Organisation

- README.md: Current file aimed towards developers who wish to use and modify the package functionality.
- contract_host.go: Defines the contract host functionality for managing contract execution environments.
- init.go: Initialization logic for the tokenomics package.

**Subpackages**

- **./contracts/**: Core contracts implementation subpackage
  - README.md: Comprehensive contracts architecture specification with behavioral patterns
  - contract.go: Main contract data structures, state machine, and lifecycle management
  - payments.go: Payment gateway interfaces and pricing models
  - proof.go: Proof and authentication mechanisms for contract verification
  - proof_verification.go: Contract proof verification logic
  - ./contracts/specs/: Contract architecture diagrams and sequence diagrams

- **./store/**: Persistent storage implementation for contracts
  - contract_store.go: CloverDB-based contract storage with upsert operations
  - contract_store_test.go: Unit tests for contract storage

- **./specs/**: Package specifications and diagrams
  - class_diagram.puml: PlantUML class diagram for tokenomics package
  - ./specs/diagrams/: Business logic and sequence diagrams

### 3. Implementation Status

The tokenomics package now contains working implementations in the contracts subpackage with contract lifecycle management, payment processing, and persistent storage.

Next steps include iterative integration with payment providers and chosen blockchain integrations, tracked by issue [Payment Layer Design and Architecture](https://gitlab.com/nunet/architecture/-/issues/502).




