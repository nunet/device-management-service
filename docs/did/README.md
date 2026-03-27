# Decentralized Identity (DID) Solutions Evaluation Report

## 1. Executive Summary

This report presents the evaluation and implementation of selected decentralized identity (DID) solutions aligned with the Cardano ecosystem and their integration into the NuNet protocol and Device Management System (DMS). The primary objective of this work was to assess the technical feasibility, integration requirements, and long-term suitability of Cardano-supported DID methods while preserving compatibility with existing NuNet architecture and capability-based authorization mechanisms.

As part of this effort, two DID methods—did:key and did:prism—were successfully integrated into the DMS. The did:key method was implemented using cryptographic keys derived from the Eternl wallet, enabling deterministic and wallet-native identity creation without introducing additional key custody or trust assumptions. In parallel, the did:prism method was integrated using a PRISM-enabled Cardano node, allowing the creation and resolution of rich, ledger-anchored DID Documents containing all required identity metadata.

In the context of the NuNet protocol, a dedicated client implementation was developed and integrated with the existing UCAN-based capability system. Rather than introducing a new identity or authorization layer, the integration extended the current capability framework to support Cardano-ecosystem DID methods. This approach ensured backward compatibility and avoided breaking changes to existing protocol components, while enabling decentralized, DID-based identity assertions and capability delegation.

The evaluation followed an implementation-driven methodology, emphasizing practical integration, standards alignment, and open-source compliance. Both DID methods were assessed in terms of security, interoperability, integration complexity, and ecosystem maturity. The results demonstrate that Cardano-supported DID methods can be incorporated into NuNet in a modular and future-proof manner, with did:key providing a lightweight and wallet-centric identity option, and did:prism enabling more expressive, ledger-anchored identity use cases.


## 2. Background & Context

### 2.1 Decentralized Identity Overview

Decentralized Identifiers (DIDs) provide a standards-based mechanism for establishing decentralized, cryptographically verifiable identities without reliance on centralized authorities. Defined by the W3C DID specification, a DID resolves to a DID Document that describes associated public keys, authentication methods, and service endpoints. These components enable secure identification, authentication, and authorization across distributed systems.

Within blockchain ecosystems, DID methods are often tightly coupled to underlying ledger infrastructures and key management models. In the Cardano ecosystem, DID approaches commonly leverage wallet-derived keys or on-chain/off-chain resolution mechanisms to ensure cryptographic integrity, interoperability, and long-term sustainability.

As part of this evaluation, particular attention is given to DID methods that are:
- Compatible with Cardano-native key management
- Aligned with open standards (W3C DID, Verifiable Credentials)
- Suitable for integration into capability-based authorization models in DMS

###  2.2 NuNet Protocol Context

NuNet is a decentralized computing framework designed to enable distributed resource sharing across environments. Identity plays a foundational role within the NuNet protocol, as it underpins node authentication, agent authorization, trust establishment, and capability delegation.

Within the NuNet context, identity must satisfy several key requirements:

- Support for decentralized, self-sovereign identifiers
- Compatibility with capability-based security models
- Non-disruptive integration with existing protocol components
- Extensibility to support future DID methods and ecosystems


As part of this work, a NuNet implementation was developed and successfully integrated with an existing UCAN-based capability system. UCAN (User Controlled Authorization Networks) provides a flexible, decentralized authorization framework that naturally aligns with DID-based identity. The integration focused on extending the capability system to support additional DID methods, rather than introducing parallel or incompatible identity mechanisms.

### 2.3 Implemented DID Methods in DMS

The Device Management System (DMS) currently supports multiple DID methods originating from the Cardano ecosystem. The following implementations were completed and validated:

#### 2.3.1 did:key Integration

The did:key method was integrated using keys derived directly from the Eternl wallet within the Cardano ecosystem. This approach ensures:

- Deterministic DID generation from existing wallet keys
- Seamless user experience by reusing established wallet infrastructure
- Strong cryptographic guarantees without additional key custody requirements

The derived did:key identifiers are used directly within the DMS and NuNet capability framework, enabling identity assertion and authorization without introducing new trust assumptions.

#### 2.3.2 did:prism Integration

The did:prism method was integrated using a PRISM-based Cardano node. DID Documents are created and resolved via the PRISM infrastructure, embedding all required identity metadata, verification methods, and service endpoints.

This approach enables:

- Ledger-anchored DID lifecycle management
- Rich DID Documents suitable for advanced identity use cases
- Alignment with emerging Cardano-native decentralized identity standards

Both DID methods were validated for interoperability within the existing DMS architecture.

### 2.4 Technical Objectives and Integration Constraints

The primary technical objective of the DID integration was to extend existing identity and authorization capabilities without introducing breaking changes. Rather than replacing or refactoring the current UCAN-based capability system, the approach focused on incremental extensibility.

Key objectives included:

- Preserving backward compatibility with existing NuNet components
- Extending the capability system to support Cardano-ecosystem DID methods
- Ensuring seamless interoperability between DIDs and UCAN capabilities
- Avoiding protocol-level disruptions or architectural regressions

By adhering to these constraints, the integration demonstrates that Cardano-supported DID methods can be incorporated into NuNet in a modular and future-proof manner, while maintaining consistency with existing security and authorization models.


## 3. Methodology

### 3.1 Evaluation Scope

The evaluation focused on decentralized identity (DID) solutions that are compatible with the Cardano ecosystem and suitable for integration within the NuNet protocol and its existing capability-based authorization framework.

The scope of the evaluation included:

- DID method compatibility with Cardano-native key management and infrastructure
- Technical feasibility of integrating DID methods into the existing DMS and NuNet client
- Alignment with UCAN-based authorization and capability delegation
- Compliance with open standards and open-source licensing requirements

### 3.2 Selection Criteria

DID solutions were evaluated against the following criteria:

#### 3.2.1 Ecosystem Compatibility

- Native or well-supported integration within the Cardano ecosystem
- Compatibility with existing wallet infrastructure (e.g., Eternl)
- Availability of Cardano-based resolution or anchoring mechanisms


#### 3.2.2 Standards Alignment

- Compliance with W3C DID specifications
- Support for Verifiable Credentials where applicable
- Extensibility toward DIDComm or equivalent secure messaging standards

#### 3.2.3 Security Model

- Cryptographic key derivation and management practices
- Support for key rotation and revocation
- Resistance to centralization or single points of failure

#### 3.2.4 Integration Complexity

- Ability to integrate without breaking changes to existing systems
- Compatibility with UCAN-based capability authorization
- API and tooling maturity

#### 3.2.5 Open-Source Maturity

- License clarity and OSI compliance
- Active development and community support
- Transparency of governance and contribution model

### 3.3 Evaluation Process

The evaluation process followed a progressive, implementation-driven approach, prioritizing practical integration over purely theoretical analysis.

#### 3.3.1 Documentation and Specification Review

- Review of W3C DID specifications and relevant extensions
- Analysis of Cardano ecosystem identity standards and proposals
- Review of PRISM architecture and resolution mechanisms
- Review of UCAN specification and existing capability flows

#### 3.3.2 Prototype Implementation in DMS

Selected DID methods were implemented directly within the Device Management System (DMS).

This included:

- Implementing did:key generation using wallet-derived keys from the Eternl wallet
- Integrating did:prism creation and resolution via a PRISM-enabled Cardano node
- Generating and resolving DID Documents with all required cryptographic material

#### 3.3.3 NuNet DMS Integration

DMS The integration process focused on:

- Mapping DID identifiers to UCAN principals
- Extending capability issuance and verification flows to support Cardano-based DID methods
- Ensuring identity assertions could be validated without protocol changes

Special care was taken to ensure:

- Backward compatibility with existing NuNet deployments
- No breaking changes to capability semantics
- Minimal coupling between DID method specifics and core NuNet logic

### 3.4 Validation and Testing

Validation was performed through functional and integration-level testing rather than isolated unit benchmarks.

Key validation activities included:

- DID creation and resolution correctness
- UCAN capability issuance and verification using DID-based principals
- Cross-component interoperability between DMS, NuNet client, and Cardano infrastructure
- Error handling and fallback behavior during DID resolution failures


## 4. Integration with NuNet Protocol

### 4.1 Identity Requirements within the NuNet Protocol

The NuNet protocol relies on decentralized identity as a foundational component for secure interaction between nodes, agents, and services in a distributed computing environment. Identity underpins authentication, authorization, capability delegation, and trust establishment across heterogeneous infrastructure.

- Key identity requirements within NuNet include:
- Decentralized, self-sovereign identity ownership
- Cryptographic verifiability of identity assertions
- Compatibility with capability-based authorization (UCAN)
- Support for multiple DID methods and key types
- Seamless interoperability with existing NuNet components
- Avoidance of breaking changes to protocol semantics

These requirements informed a modular integration approach, where identity is treated as an extensible capability rather than a tightly coupled protocol dependency.

### 4.2 UCAN-Based Capability Model Alignment

NuNet uses a UCAN (User Controlled Authorization Networks)–based capability model to manage authorization and delegation in a decentralized manner. UCAN expresses authorization as cryptographically verifiable capability chains, removing the need for centralized policy enforcement.

The DID integration aligns directly with this model by:

- Treating DIDs as first-class UCAN principals
- Using DID-associated keys for UCAN issuance and verification
- Preserving existing capability semantics and delegation logic

This alignment ensures that DID adoption enhances NuNet’s security model without introducing parallel authorization mechanisms.

### 4.3 Use of Hyperledger Identus SDK

To ensure standards-aligned and ecosystem-compatible identity generation, the integration leverages the Hyperledger Identus SDK (formerly PRISM), specifically the JavaScript/Node.js SDK.

Hyperledger Identus is an open-source decentralized identity framework designed to support W3C DIDs, Verifiable Credentials, and Cardano-native identity workflows. The Identus SDK provides cryptographic primitives and identity tooling that are compatible with both lightweight and ledger-anchored DID methods.

Within this integration:

- The Identus SDK is used to generate cryptographic key pairs
- Generated keys are used to create DIDs
- Keys and identities are exported in interoperable formats for downstream consumption by NuNet components

This approach ensures strong alignment with open standards while maintaining compatibility with Cardano ecosystem tooling.

### 4.4 Cross-Language Key and Identity Portability (JS → Go)

A key architectural requirement was seamless interoperability between:

- JavaScript-based identity tooling (Identus SDK)
- The Go-based NuNet Device Management Service (DMS)

To bridge this gap, keys generated via the Identus SDK are exported in standardized formats, including:

- JSON Web Key (JWK) (RFC 7517)
- Raw hexadecimal key material

The DMS is designed to import these keys directly, allowing identities generated in a JavaScript environment to be used natively within NuNet’s Go-based runtime.

At the current milestone:

- Secp256k1 keys generated via Identus can be imported into DMS using raw hexadecimal format
- A full JWK-based importer is under active implementation
- Imported keys are compatible with existing libp2p cryptographic primitives used by NuNet

This design ensures that identity generation and management remain flexible, language-agnostic, and future-proof.

### 4.5 did:key Integration in NuNet

The did:key method is supported as a lightweight, infrastructure-independent identity mechanism.

Key characteristics include:

- DID generation derived directly from cryptographic keys
- No dependency on ledger interaction or external resolution infrastructure
- Immediate compatibility with UCAN-based capability verification

In practice:

- Keys generated via the Identus SDK or derived from Cardano wallets (e.g., Eternl) are used to construct did:key identifiers
- DID resolution is performed locally
- Public keys from the resolved DID Document are used directly for UCAN verification

This makes did:key particularly suitable for agent-level identities, ephemeral nodes, and high-frequency interactions within NuNet.

### 4.6 did:prism Integration in NuNet

The did:prism method is integrated using PRISM-compatible tooling and Cardano infrastructure, enabling ledger-anchored decentralized identities.

Key aspects include:

- DID creation and resolution via PRISM-based Cardano nodes
- Support for rich DID Documents, including verification methods and service endpoints
- Explicit identity lifecycle management anchored to the Cardano ledger

Within NuNet:

- did:prism identifiers are resolved using PRISM infrastructure
- The resolved DID Documents provide the cryptographic material required for UCAN verification
- These identities are well-suited for long-lived nodes, infrastructure components, and governance-sensitive roles


### 4.7 Non-Breaking Integration Strategy

A strict constraint of the integration was to avoid breaking changes to the existing NuNet protocol and capability system.

This was achieved by:

- Extending identity abstractions rather than replacing them
- Introducing DID support as an additive feature
- Maintaining backward compatibility with existing principals
- Avoiding DID-method-specific logic within core NuNet components

As a result, existing NuNet deployments can incrementally adopt DID-based identities without mandatory migrations or protocol upgrades.


### 4.8 Operational and Implementation Considerations

The integration surfaced several practical considerations:

- Key Management: Wallet-derived and SDK-generated keys simplify onboarding but require clear rotation and revocation strategies.
- Resolution Dependencies: did:prism introduces reliance on PRISM node availability, whereas did:key remains fully self-contained.
- Performance Trade-offs: Local resolution (did:key) offers lower latency compared to ledger-anchored resolution (did:prism).
- Developer Experience: Identus SDK provides a consistent, standards-aligned developer interface across identity use cases.

These considerations inform the phased roadmap and adoption recommendations presented later in this report.


### 4.9 Summary

The integration demonstrates that Cardano-ecosystem DID methods can be incorporated into the NuNet protocol using open-source, standards-compliant tooling without disrupting existing architecture. By leveraging the Hyperledger Identus SDK for key and identity generation, and enabling cross-language portability into the Go-based DMS, NuNet gains a flexible and extensible identity foundation aligned with both UCAN capabilities and Cardano-native decentralized identity standards.

## 5. Roadmap: Decentralized Identity Integration for NuNet

This roadmap summarizes the phased implementation of decentralized identity support within the NuNet ecosystem, highlighting completed milestones and remaining integration tasks. The approach emphasizes Cardano ecosystem compatibility, UCAN capability alignment, and non-breaking architectural evolution.

### 5.1 Milestone 1: did:key Integration with Cardano Wallets (Completed)
Objective: Enable lightweight, wallet-native decentralized identities using Cardano ecosystem tooling.
Key Achievements: 

- Integrated did:key support into the Device Management System (DMS)
- Derived cryptographic keys directly from the Eternl wallet
- Enabled deterministic DID generation without additional key custody
- Validated seamless use of did:key identities within UCAN-based capability flows

Outcome:
Established a low-latency, infrastructure-independent identity mechanism suitable for agents and ephemeral NuNet components.

### 5.2 Milestone 2: did:prism Integration (Completed)


Objective:  Support richer, ledger-anchored decentralized identities aligned with Cardano-native standards.

Key Achievements: 

- Integrated did:prism support using a PRISM-enabled Cardano node
- Enabled creation and resolution of DID Documents containing required identity metadata
- Integrated ledger-anchored identities into DMS and NuNet capability validation
- Preserved backward compatibility and avoided protocol-level changes

Outcome: Enabled long-lived, governance-aware identities suitable for NuNet nodes and infrastructure roles.

### 5.3 Milestone 3: Hyperledger Identus SDK Key and Identity Generation (Completed)

Objective: Standardize key and identity generation using open-source, ecosystem-aligned tooling.
Key Achievements:

- Adopted the Hyperledger Identus JavaScript SDK for cryptographic key generation
- Generated Secp256k1 keys compatible with NuNet and DMS
- Exported keys in interoperable formats (raw hex and JWK)
- Demonstrated cross-language portability (JavaScript → Go)
- Successfully imported raw key material into the Go-based DMS

Outcome: Established a standards-aligned, language-agnostic identity generation workflow compatible with NuNet infrastructure.

### 5.4 Next Step: JWK Import Support in DMS

Remaining Work:

- Implement full JWK private key import support in the DMS
- Enable direct consumption of Identus-generated JWKs without raw key fallback
- Unify key import paths across formats

### Delivered

#### Milestone 1: Eternl Integration

Milestone https://milestones.projectcatalyst.io/projects/1400088/milestones/1 delivered

Code Merged: https://gitlab.com/nunet/dev-tools/eternl-cli


#### Milestone 2: did:prism Integration 

Milestone https://milestones.projectcatalyst.io/projects/1400088/milestones/2 delivered

Code Merged: https://gitlab.com/nunet/device-management-service/-/merge_requests/1215

#### Milestone 3: Hyperledger Identus SDK

Milestone https://milestones.projectcatalyst.io/projects/1400088/milestones/3 delivered

Code Merged: https://gitlab.com/nunet/device-management-service/-/merge_requests/1222

### Roadmap Summary

The identity integration roadmap demonstrates a clear progression from wallet-based identities (did:key), to ledger-anchored identities (did:prism), and finally to standardized, SDK-driven key and identity generation using Hyperledger Identus. The remaining JWK import functionality represents a focused, well-defined final step toward fully standardized identity interoperability within the NuNet Device Management System.