
- [Milestone: Device Management Service Version 0.5.x](#milestone-device-management-service-version-05x)
  - [Release management](#release-management)
    - [Branching strategy](#branching-strategy)
    - [Release process](#release-process)
      - [Feature scope freeze](#feature-scope-freeze)
      - [Specification and documentation portal launch](#specification-and-documentation-portal-launch)
    - [Project management portal launch](#project-management-portal-launch)
    - [Test management system launch](#test-management-system-launch)
    - [Release testing start](#release-testing-start)
    - [Release](#release)
  - [Feature scope](#feature-scope)
    - [Infrastructure](#infrastructure)
      - [Feature environment](#feature-environment)
      - [CI/CD pipeline](#cicd-pipeline)
      - [Test management](#test-management)
    - [Documentation](#documentation)
      - [Specification and documentation system](#specification-and-documentation-system)
      - [Project management portal](#project-management-portal)
    - [Implementation](#implementation)
      - [Actor model with Object/Security Capabilities](#actor-model-with-objectsecurity-capabilities)
      - [Object/Security Capabilities (UCAN)](#objectsecurity-capabilities-ucan)
      - [Dynamic method dispatch/invocation](#dynamic-method-dispatchinvocation)
      - [Local access (API and CMD)](#local-access-api-and-cmd)
      - [Local database interface and implementation](#local-database-interface-and-implementation)
      - [Executor interface and implementation](#executor-interface-and-implementation)
      - [Machine benchmarking](#machine-benchmarking)
      - [p2p network and routing](#p2p-network-and-routing)
      - [Storage interface](#storage-interface)
      - [IP over Libp2p](#ip-over-libp2p)
      - [Observability and Telemetry](#observability-and-telemetry)
      - [Definition of compute workflows/recursive jobs](#definition-of-compute-workflowsrecursive-jobs)
      - [Job deployment and orchestration model](#job-deployment-and-orchestration-model)
      - [Hardware capability model](#hardware-capability-model)
      - [Supervision model](#supervision-model)
      - [Tokenomics interface](#tokenomics-interface)



# <a id="milestone-device-management-service-version-05x"></a>Milestone: Device Management Service Version 0.5.x

Context / links:
1. [Project management portal page for the milestone](https://docs.nunet.io/project-management-portal/device-management-service-version-0-5-x)
2. [Technical dependencies / work packages graph](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages_technical_dependencies.html)
3. [Milestone goals / description](https://gitlab.com/nunet/architecture/-/issues/182)
3. [Tracking issue](https://gitlab.com/nunet/device-management-service/-/issues/519)
3. [DMS 0.5.0 - feature scope GitLab board](https://gitlab.com/groups/nunet/-/boards/7786612?milestone_title=Device%20Management%20Service%20Version%200.5.x)
4. [DMS 0.5.0 - release process GitLab board](https://gitlab.com/groups/nunet/-/boards/7762598?milestone_title=Device%20Management%20Service%20Version%200.5.x)


## Release management

The goal of the process is to expose all developed functionalities with frozen scope, as described above, to a rigorous internal and community testing via succession of release candidate builds. 

### Branching strategy

In preparation for this release process we will be changing the branching strategy as discussed and agreed in this [issue](https://gitlab.com/nunet/device-management-service/-/issues/542) and tracked by this [issue](https://gitlab.com/nunet/device-management-service/-/issues/547). In summary:

* We will operate on two branches: `main` (for trunk development) and `release` (for minor / final releases and patches).
* The public dms binaries will be built from `release` branch tags in the form of v{version_number}-{suffix}, e.g. `v0.5.0-rc1`, etc. and released via [gitlab release page](https://gitlab.com/nunet/device-management-service/-/releases) as usual.
* Interim builds between scheduled release candidates will be tagged with suffix `-p{patch_number}`, e.g. `v0.5.0-rc1-p1`, etc.

### Release process

The release process of [Device Management Service Version 0.5.x](https://gitlab.com/groups/nunet/-/milestones/44#tab-issues) is scheduled to start September 15, 2024 and finish December 15, 2024. It involves the following steps in chronological order:

1. Feature scope freeze
2. Specification and documentation system launch
3. Project management portal launch 
4. Test management system launch (feature environment, CI/CD pipeline and QA visibility)
8. Release candidates testing and delivering full feature scope

The currently scheduled release candidates are:

1. `v0.5.0-boot` -- bootstrap release;
2. `v0.5.0-rc1` -- first release candidate;
3. `v0.5.0-rc2` -- second release candidate;
4. `v0.5.0-rc3` -- third release candidate;
5. `v0.5.0-rc4` -- fourth release candidate;
6. `v0.5.0` -- final release

The internal latest updated document with release process dates and countdown is available to view [here](https://docs.google.com/spreadsheets/d/1fIileKjI1SqD_9Xv-jHu0gm8cY2y689xl8KwqkIXu44/edit?gid=0#gid=0). Feature scope management is done via [DMS 0.5 pubic issue board](https://gitlab.com/groups/nunet/-/boards/7762598?milestone_title=Device%20Management%20Service%20Version%200.5.x).

#### Feature scope freeze

We first define the final scope of the dms features that we are releasing within this milestone, which is done within [this section](#implementation) of the document. The scope of each feature should defined by functional test cases / scenarios associated with each feature and linked in **Acceptance tests** row of each table. All required test cases and scenarios pertaining to the feature scope freeze shall be written by the time of the last last release candidate release. The work is tracked by the work package [dms/tests](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/tests_technical_dependencies.html) integrated into the test management system. Until then we will be continuing to implement all the features as they are explained in this document.

#### Specification and documentation portal launch

Documentation portal was operating internally since end of 2023, but was not fully aligned with the specification and documentation process that involves the whole team and including updates of documentation into acceptance criteria of each pull request. Documentation portal shall be launched in August. The work involving finishing and launching documentation portal are tracked by [this issue](https://gitlab.com/nunet/publisher/documentation/-/issues/4). 

### Project management portal launch

Project management portal has been operating internally since Q4 2023, but was not exposed publicly as of yet. It will be launched publicly by the end of August 2024. The work involved in finishing and launching the project management portal for the dms v.0.5 milestone is tracked by [this issue](https://gitlab.com/nunet/publisher/project-management-portal/-/issues/17)

### Test management system launch

NuNet test management system consists of three main components, which have been developed for months now and are close to completion, but need to be finally coordinated and aligned in the preparation for the start of the DMS v0.5 release process and release candidate testing internally as well as via the community contributions. The components of the test management system are:

1. CI/CD pipeline that publishes and makes available for further publishing all testing artifacts and reports; These are further used by developers, QA engineers and shall be exposed to community developers publicly; 
2. Feature environment features a small network of geographically distributed virtual machines connected via public internet, and allows for the execution of selected CI/CD pipeline stages automatically on heterogenous hardware environments -- testing functionality of the fully built DMS; Most of the acceptance tests of the frozen feature scope will be running via the 
3. QA management portal is a web portal (Testmo) which exposes all test artifacts via single interface and provides all information for NuNet QA team to see which test are passing / failing for every built of the DMS.  

All three components are tightly coupled and will be internally released in quick succession, the last week of August - first week of September, targeting finalizing the test management system launch at the end of first week of September, so that the quality of release candidates can be fully validated by QA team.

Both feature environment and CI/CD process are instrumental to reach release level quality of the developed features. Both have been in development for months prior to current moment and are close to completion. Both are moving targets as they will develop together with the platform. Nevertheless, we aim to launch them as described above in the first half of August.

### Release testing start

Given that all prerequisites are launched and prepared, we aim at starting the release process September 15, 2024 by releasing the first release candidate and exposing it to testing. We will release at least 3/4 release candidates and then final release, but the process will be fluid as explained [here](https://gitlab.com/nunet/device-management-service/-/issues/542) and most probably fill feature more minor release candidate releases per the adopted branching strategy.

### Release

After testing the frozen feature scope of the release, we aim at releasing the 0.5.0 version of device-management-service during the second half of December 2024. For current / updated schedule and details, see [release process kick-start presenatation](https://docs.google.com/presentation/d/1lS6sI7-v5QyFf1xsM-y-SK8tHJ0rEzpjBQpb2lPQ7yI/edit#slide=id.g2f39a3f1c57_0_5) and [constantly updated countdown document](https://docs.google.com/spreadsheets/d/1fIileKjI1SqD_9Xv-jHu0gm8cY2y689xl8KwqkIXu44/edit?gid=0#gid=0).

## Feature scope

### Infrastructure

#### Feature environment

| | |
| --- | --- |
| **Feature name** | Bootstrap feature environment composed of geographically distributed machines connected via public internet |
| **Work package** | [deliverables](https://gitlab.com/nunet/architecture/-/issues/208) <br>[tech dependencies](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/ci-cd-pipeline_technical_dependencies.html) |
| **Code reference** | https://gitlab.com/nunet/test-suite/-/tree/develop/environments/feature?ref_type=heads |
| **Description / definition of done** | 1) Selected stages of CI/CD pipeline are automatically executed on the fully functional feature environment consisting of geographically dispersed virtual machines<br>2) Developers are able to test functionalities requiring more than one machine running code;<br>3) testing artifacts are collected and made accessible from centralized place (GitLab, Testmo, etc) for all tested scenarios / configurations and machines.<br>4) Interface / process for configure and manage feature environment configurations and reason about complex scenarios involving multiple machines   | 
| **Timing** | 1) Capability exposed to development team as soon as possible along in the preparation of release process (could comply to definition of done partially); 2) Full definition of done met and all impacted funcionalities considered by the September 1, 2024 at the latest |
| **Status** | On track  |
| **Team** | Lead: @gabriel.chamon; Supporting: @sam.lake, @abhishek.shukla3143438 |
| **Strategic alignment** | 1) Enables advanced testing and quality assurance on limited hardware; 2) Enables scaling the testing environment from internally onwed to community developers machines, and eventually into testnet; <br>3) Builds grounds for release management now and in the future (including release candidates, testnet configurations involving third party hardware, canary releases, etc) |
| **Who it benefits** | 1) NuNet development team ensuring next level quality of CI/CD process and released software <br>2) Community testers and early compute providers <br>3) Solution providers who are already considering using NuNet for their business processes and will start testing in release candidate phase<br>4) Eventually all platform users  will benefit from the quality of software |
| **User challenge** | 1) Developers want to test software as early as possible and access testing data / different hardware / software configurations in real network setting;<br> 2) Community compute providers want to contribute to the development by providing machines for testnet  |
| **Value score** | n/a |
| **Design** | n/a |
| **Impacted functionality** | This does not affect any feature (except possibly ability to launch testnet in the future) but rather deals with quality assurance of the whole platform, therefore indirectly but fundamentally affects quality of all features now and in the future.|
| **Acceptance tests** | Developers are able to spawn and tear down testing networks on demand for testing custom platform builds and complex functionalities involving more than one machine; CI/CD pipeline incorporates complex functional and integration tests which run on automatically spawned and teared down small (but geographically distributed) real networks of virtual machines |

#### <a id="#cicd-pipeline"></a>CI/CD pipeline

| | |
| --- | --- |
| **Feature name** | Set the CI/CD pipeline able to run full test suite on all environments as defined in text matrix |
| **Work package** | [deliverables](https://gitlab.com/nunet/architecture/-/issues/208) <br>[tech dependencies](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/ci-cd-pipeline_technical_dependencies.html) |
| **Code reference** | https://gitlab.com/nunet/test-suite/-/tree/develop/cicd?ref_type=heads |
| **Description / definition of done** | 1) CI/CD pipeline is structured as per general test management and test matrix based-approaches and executes the minimal configuration for this milestone; 2) The CI/CD process explained and documented so that it could be evolved further via developers contributions <br>3) The minimal configuration is: <br>3.1) unit tests run per each defined test matrix trigger / VCS event;  <br>3.2) functional tests (the ones that require code build, but no network) are executed per each defined test matrix trigger / VCS event <br>4) Each stage of CI/CD pipeline collects and publishes full test artifacts for <br>4.1) developers for easy access via GitLab flow; <br>4.2) code reviewers for easy access via GitLab pull requests; <br>4.3) QA team via Testmo interface; <br>5) QA requirements and availabilities provided by CI/CD pipeline are integrated into issue Acceptance Criteria | 
| **Timing** | 1, 2, 3 implemented within first half of August 2024; 4 implemented within August; 5 implemented as soon as all is finished, but not later than first week of September; |
| **Status** | On track |
| **Team** | Lead: @gabriel.chamon; Supporting: @abhishek.shukla3143438, @ssarioglunn |
| **Strategic alignment** | 1) CI/CD pipeline integrates to the development process by enabling automatic testing of code against formalized functional requirements / scope of each release / milestone; <br>2) CI/CD pipeline process integrates into the test management process (of integrating new tests into the pipeline and ensuring that all testing domain is covered with each build) <br>3) Proper CI/CD pipeline and its visibility is instrumental for reaching our business goals of delivering quality software that is evolvable  |
| **Who it benefits** | 1) NuNet development team ensuring next level quality of CI/CD process and released software <br>2) Internal QA team having the interface to easily access all built errors / issues and update requirements as needed <br>3) Eventually all platform users  will benefit from the quality of software |
| **User challenge** | device-management-service code with the same build will run in diverse scenarios and diverse machine / os configurations while providing robust functionality; CI/CD pipeline process (together with test management process) will need to ensure that <br>a) all QA requirements for implemented functionality scope are properly formalized and placed in correct test stages for exposing each build to them <br>b) updated and/or newly defined functionality scope (during release process), including tests included as a result of community bug reports, are properly integrated into QA scope abd the CI/CD pipeline |
| **Value score** | n/a |
| **Design** | n/a |
| **Impacted functionality** | This does not affect any feature (except possibly ability to launch testnet in the future) but rather deals with quality assurance of the whole platform, therefore indirectly but fundamentally affects quality of all features now and in the future.|
| **Acceptance tests** | n/a |

#### Test management

| | |
| --- | --- |
| **Feature name** | Set up test management system, visibility tools and integration with CI/CD and Security pipelines |
| **Work package** | [deliverables](https://gitlab.com/nunet/architecture/-/issues/207) <br>[tech dependencies](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/test-management-system_technical_dependencies.html) |
| **Code reference** | https://gitlab.com/nunet/test-suite |
| **Description / definition of done** | As detailed in work package deliverables: https://gitlab.com/nunet/architecture/-/issues/207 |
| **Status** | With challenges |
| **Team** | Lead: @abhishek.shukla3143438; Supporting: @gabriel.chamon, @ssarioglunn |
| **Strategic alignment** | 1) Test management system will enable <br>1.1) Define quality requirements and functionalities formally <br>1.2) Release software that precisely corresponds to the defined quality / functional requirements; <br> 2) Test management system will enable for the functional and quality requirements to evolve together with the code; <br> 3) Test management system implementation will enable for the community developers and testers to be part of the development process and greatly enhance it; <br> 4) Participation of community in test management process will ideally be related to contributors program |
| **Who it benefits** | 1) NuNet development team ensuring next level quality of the released software <br>2) Internal QA team having the interface to easily access all built errors / issues and update requirements as needed <br>3) Eventually all platform users  will benefit from the quality of software |
| **User challenge** | 1) Test management process should be flexible to evolve together with the code and constantly get updated with test vectors / scenarios; These can come both from core team and community developers / testers; Our test management system is fairly complex, involving different stages and frameworks (go for unit tests; Gherkin for other stages) as well as multiple machine environments, simulating real network behavior. In order to reach our goals, careful coordination (fitting together) of all aspects is needed. <br> 2) Special challenge is inclusion of manual tests into the same framework of visibility, but enabling QA team to coordinate manual testing schedules and inclusion of their results into the test artifacts; <br> 3) Community developers and testers should be included into the testing process both by public access to the test suite definitions as well as manual testing process, which is related to contributors programme; |
| **Value score** | n/a |
| **Design** | as described https://gitlab.com/nunet/test-suite |
| **Impacted functionality** | This does not affect any feature (except possibly ability to launch testnet in the future) but rather deals with quality assurance of the whole platform, therefore indirectly but fundamentally affects quality of all features now and in the future.|
| **Acceptance tests** | n/a |


### Documentation

#### Specification and documentation system

| | |
| --- | --- |
| **Feature name** | Set up specification and documentation system for all nunet repos |
| **Work package** | [tech dependencies](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/data-model-specification-documentation-system_technical_dependencies.html) |
| **Code reference** | [documentation portal](https://gitlab.com/nunet/publisher/documentation) <br>[important issue related to README.md](https://gitlab.com/nunet/team-processes-and-guidelines/-/issues/5) <br>[MR converting marmaid to plantuml, containing description of the process](https://gitlab.com/nunet/device-management-service/-/merge_requests/373) |
| **Description / definition of done** | 1) The documentation process is set up and described, including <br>1.1) requirement to have README.md files in each package and subpackage, <br>1.2) general structure and required contents of README.md files; <br>2) All packages and subpackages contain class diagram written in plantuml and their hierarchy is correctly constructed; <br>3) Documentation system allows to include package / subpackage specific descriptions or extensions via links in README.md and possibly using additionally files (also in ./specs directory) as needed; <br> 4) Updating specification and documentation of respective packages is included into acceptance criteria of each issue and checked during merge request reviews; <br>5) Documentation system includes process of requesting and proposing new functionalities and architecture; <br>6) Regular builds of documentation portal which is based on README.md files |
| **Status** | Close to completion |
| **Team** | Lead: @0xPravar; Supporting: @kabir.kbr, @janaina.senna |
| **Strategic alignment** | As explained in [tech discussion 2024/07/25](https://docs.google.com/presentation/d/1NyLo3W0Ee9ygyvYSYr5jVZdTwYmbITdO_vRVwneZmKM/edit#slide=id.g1fcbec5b67a_0_184) the goals of specification and documentation system are: <br>1. Clarity and understanding of:<br>1.1 architecture; <br>1.2 functionality; <br>1.3 tests [i.e. functionality with some emphasis on quality]; <br>2. [by that enabling/helping] coordination [of development activities] by: <br>2.1 the core team; <br>2.2 community; <br> 2.3 integration partners; <br> 3. Making documentation part of the code, i.e. living documentation, which changes with the code and reflects both current status and planned / proposed directions;|
| **Who it benefits** | 1) NuNet technical board by having space where conceptual architecture and priorities are translated into architectural and functional descriptions <br>2) Internal platform development team as well as community developers have clearly defined structure architecture and proposed functionality guiding the work and collaboration <br>4) Internal integrations team (developing use-cases) and third party solution providers have clear documentation of the platform to build solutions on top <br>5) integration partners, using the system can understand the current and future integration potential on a deep technical level and propose required functionality directly into the development process |
| **User challenge** | In order for all stakeholder groups to benefit, the documentation and specification system should be <br> 1) structured enough for all stakeholders to quickly find aspects that are specifically needed for them; <br>2) flexible enough to express package architecture and implementation specific for maximal clarity and understanding; <br>3) integrated into development process for easy access by and contribution from internal development team<br> 4) conveniently accessible for communication to community developers, integration partners (for this milestone) and other stakeholders from broader ecosystem (in the future);   |
| **Value score** | n/a |
| **Design** | as described during [tech discussion 2024/07/25](https://docs.google.com/presentation/d/1NyLo3W0Ee9ygyvYSYr5jVZdTwYmbITdO_vRVwneZmKM/edit?usp=sharing) |
| **Impacted functionality** | This does not affect any feature directly but fundamentally enables the quality and integrity of the whole platform functionality, alignment with the business goals, use-case based platform development model and evolvability of the software.|
| **Acceptance tests** | [documentation portal launch issue and deliverables](https://gitlab.com/nunet/publisher/documentation/-/issues/4) |


#### Project management portal

| | |
| --- | --- |
| **Feature name** | Prepare and launch project management portal |
| **Work package** | There is not specific work package in the current milestone, as project management portal was developed outside its scope; |
| **Code reference** | [Drive Folder](https://drive.google.com/drive/folders/1G8bGHW8ORVSapEqJDFh43oGP8UOQu15C) with related working documents and discussions; <br> [project management portal repository](https://gitlab.com/nunet/publisher/project-management-portal); <br>[gitbook team pages](https://app.gitbook.com/o/HmQiiAfFnBUd24KadDsO/s/UoAe47nbLYU9lZHCUSJH/); <br>[gitbook public pages](https://docs.nunet.io/project-management-portal) |
| **Description / definition of done** | 1) Project management pages always reflect the latest status of each milestone; <br>2) Introductory page explains project management portal and process for internal and external users <br>|
| **Status** | Close to completion |
| **Team** | Lead: @janaina.senna; Supporting: @0xPravar, @kabir.kbr |
| **Strategic alignment** | Project management portal is a part of the Critical Chain Project management process that is the basis of our development flow and is integrated into the team process; The aim of CCPM process is to achieve the level where we are able to plan, schedule and implement milestones and releases without delay; The portal is designed to automatically generate and share all important updates about the process with the team as well as external contributors in order to make the process as efficient and transparent as possible; The process is used internally since mid 2023 and shall be included into the contribution program and community developers process;  |
| **Who it benefits** | 1) Development team follows the CCPM process via the project management portal; <br>2) Marketing team uses project management portal (as well as other internal resources) to shape communications transparently; <br>3) Operations and business development team can always see the recent status in development process |
| **User challenge** |    |
| **Value score** | n/a |
| **Design** | As described in [project management portal repository](https://gitlab.com/nunet/publisher/project-management-portal) readme |
| **Impacted functionality** | This does not affect any feature directly but fundamentally enables alignment with the business goals, use-case based platform development model and evolvability of the software.|
| **Acceptance tests** | [Project management portal launch issue and deliverables](https://gitlab.com/nunet/publisher/project-management-portal/-/issues/17) |

### Implementation

#### <a id="#actor-model-with-objectsecurity-capabilities"></a>Actor model with Object/Security Capabilities

| | |
| --- | --- |
| **Feature name** | Actor model and interface; Node and Allocation actors implementations; [feature::actor-model](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aactor-model&first_page_size=20); [feature::general](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Ageneral&first_page_size=20)|
| **Work packages** | [dms](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/dms-package-implementation_technical_dependencies.html) <br>[node](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/node-package-implementation_technical_dependencies.html) <br>[jobs](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/jobs-package-implementation_technical_dependencies.html) <br>[network](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/network-package-implementation_technical_dependencies.html)|
| **Code reference** | https://gitlab.com/nunet/test-suite/-/tree/develop/environments/feature?ref_type=heads |
| **Description / definition of done** | 1) Machine onboarded on NuNet via dms act as separate actors (via Node interface); <br>2) Jobs deployed and orchestrated on the platform act as separate Actors (via Allocation interface); <br>3) Nodes and Allocations (both implementing the Actor interface) communicate only via the immutable messages (via Message interface and network package's transport layer) and have no direct access to each other private state;| 
| **Timing** | Actor, Node and Allocation interfaces are correctly implemented at the start of the release process; correct interface implementation means that the send messages to each other, can read and process received messages and initiate arbitrary behaviors via RPC model;  |
| **Status** | With challenges |
| **Team** | Lead: @mobin.hosseini; Supporting: @gugachkr |
| **Strategic alignment** | 1) Enables the unlimited horizontal scalability of the platform with potentially minimal manual intervention; <br>2) Enables any eligible entity to join the platform without central control point; <br>3) Enables concurrency and non-locking by design which is independent of the scale of the platform at any point in time |
| **Who it benefits** | 1) Solution integrators (providers) who need to use Decentralized Physical Infrastructure; <br>2) Hardware owners who wish to utilize their infrastructure in a more flexible manner than possible with established orchestration and deployment technologies and without direct involvement with the compute users or building compute marketplaces |
| **User challenge** | 1) All compute users want to maximize efficient and fast access to optimal hardware for doing a computing job at hand and not to overpay for that; <br>2) Hardware resource owners want the maximal utilization of their hardware resources without idle time; |
| **Value score** | n/a |
| **Design** | [DMS architecture](https://nunet.gitlab.io/research/blog/posts/dms-architecture/) |
| **Impacted functionality** | All functionality of the platform is fundamentally affected implementation of actor model; This is especially true for the future projected functionalities involving edge computing, IoT deployments and decentralized physical infrastructure in general. |
| **Acceptance tests** | Functional and integration tests defined in node package, dms package related to Actor interface and jobs package related to Allocation interface; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/122) |

#### <a id="#objectsecurity-capabilities-ucan"></a>Object/Security Capabilities (UCAN)

| | |
| --- | --- |
| **Feature name** | Implementation of User Controlled Authorization Network (UCAN); DIDs and key management; [feature::ucan](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aucan&first_page_size=20) |
| **Work packages** | [actor](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/actor-package-implementation_technical_dependencies.html) |
| **Code reference** |  |
| **Description / definition of done** | | 
| **Timing** | Closely integrated with the Actor system implementation; Every message requires UCAN tokens to be included and verified;  |
| **Status** | Close to completion |
| **Team** | Lead: @vyzo |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
 |
| **Impacted functionality** | Implementation of the fundamental zero trust security model. |
| **Acceptance tests** |  |

#### <a id="#dynamic-method-dispatchinvocation"></a>Dynamic method dispatch/invocation

| | |
| --- | --- |
| **Feature name** | Dynamic method dispatch logic for initiating behaviors in actors; [feature::remote-invocation](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aremote-invocation&first_page_size=20) |
| **Work packages** | Implemented within the scope of [Node package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/node-package-implementation_technical_dependencies.html)  |
| **Code reference** | [issue](https://gitlab.com/nunet/device-management-service/-/issues/474) with minimal description; |
| **Description / definition of done** | Methods / functions can be run remotely by sending a message from one Actor to another | 
| **Timing** |  |
| **Status** | In progress |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** | |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  |
| **Impacted functionality** | Fundamental functionality that enables the full realization of the Actor model potential |
| **Acceptance tests** | Unit tests of around 90%; Functional / integration tests: sending rpc call from one actor (node or allocation) on different network configuration to another Actor (node or allocation); and initiate chosen method; ; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/137) |

#### <a id="#local-access-api-and-cmd"></a>Local access (API and CMD)

| | |
| --- | --- |
| **Feature name** | Local access to running dms from the machine on which it is running; [feature::cli-access](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Acli-access&first_page_size=20) |
| **Work packages** | [api work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/api-package-implementation_technical_dependencies.html); <br>[cmd work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/cmd-package-implementation_technical_dependencies.html)  |
| **Code reference** | [api package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/api?ref_type=heads) <br>[cmd package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/cmd?ref_type=heads) |
| **Description / definition of done** | [api package deliverables](https://gitlab.com/nunet/architecture/-/issues/243); <br>[cmd package deliverables](https://gitlab.com/nunet/architecture/-/issues/229) | 
| **Timing** |  |
| **Status** | Almost complete |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** | |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  |
| **Impacted functionality** | Configuration of dms; Access to NuNet network from external applications via REST-API; |
| **Acceptance tests** | Unit tests of around 90%; Functional / integration tests: api responds to locally issued commands; api does not respond to remotely issued commands; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/123) |

#### Local database interface and implementation

| | |
| --- | --- |
| **Feature name** | Local NoSQL database interface and implementation; [feature::local-db](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Alocal-db&first_page_size=20) |
| **Work packages** | [database work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/database-package-implementation_technical_dependencies.html)  |
| **Code reference** | [database package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/db?ref_type=heads) |
| **Description / definition of done** | [database work package deliverables](https://gitlab.com/nunet/architecture/-/issues/227)  | 
| **Timing** |  |
| **Status** | Almost completed |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  |
| **Impacted functionality** | Configuration management; Local telemetry and logging management (possibly); |
| **Acceptance tests** | Unit tests of around 90%; Functional / integration tests: Arbitrary information can be stored, retrieved and searched via the implemented interface; ; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/136) |

#### Executor interface and implementation

| | |
| --- | --- |
| **Feature name** | Executor interface and implementation of docker and firecracker executors; [feature::execution-and-resources](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aexecution-and-resources&first_page_size=20)|
| **Work packages** |  |
| **Code reference** | [executor package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/db?ref_type=heads) |
| **Description / definition of done** | [executor design work package deliverables](https://gitlab.com/nunet/architecture/-/issues/233) <br>[executor implementation work package deliverables](https://gitlab.com/nunet/architecture/-/issues/232)  | 
| **Timing** |  |
| **Status** | Finished |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  |
| **Impacted functionality** | Definition of generic interface for easy plugging third party developed executables to dms; Full implementation of docker and firecracker executables; |
| **Acceptance tests** | Unit tests of around 90%; Functional / integration tests: starting a compute job with docker / firecracker executables; observing the runtime; finishing and receiving results;  ; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/135)|

#### Machine benchmarking

| | |
| --- | --- |
| **Feature name** | Machine [Capability] benchmarking ; [feature::machine-benchmarking](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Amachine-benchmarking&first_page_size=20) |
| **Work packages** | [node work package](https://gitlab.com/nunet/architecture/-/issues/346) |
| **Code reference** | DMS package code and subpackages (mostly [node](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/node?ref_type=heads), [onboarding](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/onboarding?ref_type=heads) and [resources](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/resources?ref_type=heads) subpackages) |
| **Description / definition of done** | <br>[relevant issue with minimal description](https://gitlab.com/nunet/benchmarking/-/issues/1)  | 
| **Timing** |  |
| **Status** | In progress |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  |
| **Impacted functionality** | Machine benchmarking is needed for the Capability / Comparison interface implemented in [dms.orchestrator.matching](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/orchestrator/matching?ref_type=heads) subpackage |
| **Acceptance tests** | Unit tests; Functional tests: Machines are benchmarked while onboarding, the benchmarking data is stored / accessed via database interface; ; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/134)|


#### p2p network and routing

| | |
| --- | --- |
| **Feature name** | p2p network and routing; [feature::p2p-network](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Ap2p-network&first_page_size=20) |
| **Work packages** | [network work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/network-package-implementation_technical_dependencies.html) |
| **Code reference** | [network package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/network?ref_type=heads) |
| **Description / definition of done** | implemented [network package design](https://gitlab.com/nunet/architecture/-/issues/241) | 
| **Timing** |  |
| **Status** | Close to completion |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Sending and receiving messages directly or via broadcasts; |
| **Value score** | n/a |
| **Design** | Messages and routing partially explained in research blog on [gossipsub, DHT and pull/push mechanisms](https://nunet.gitlab.io/research/blog/posts/gossipsub/)  |
| **Impacted functionality** | Fundamental functionality of NuNet -- connecting dms's into p2p neworks and subnetworks; |
| **Acceptance tests** | Unit tests; Functional tests: Actors (nodes and allocations) are able to see peers / neighbours; It is possible to send and receive messages from other Actors (nodes and allocations) either directly (addressed) or via gossip routing indirectly; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/133) |

#### Storage interface

| | |
| --- | --- |
| **Feature name** | Storage interface definition and s3 storage implementation; [feature::storage](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Astorage&first_page_size=20)  |
| **Work packages** |  |
| **Code reference** | [storage package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/storage?ref_type=heads) |
| **Description / definition of done** | [storage package design work package description](https://gitlab.com/nunet/architecture/-/issues/254); <br>[storage package implementation work package description](https://gitlab.com/nunet/architecture/-/issues/255) | 
| **Timing** |  |
| **Status** | Finished |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Ability for running executors to access data via storage interface and its implementations |
| **Value score** | n/a |
| **Design** |  [list of related issues and comments](https://gitlab.com/search?search=storage&nav_source=navbar&project_id=23187917&group_id=6160918&scope=issues)  |
| **Impacted functionality** | Fundamental functionality of NuNet -- providing input and output data storage for computation processes |
| **Acceptance tests** | Unit tests; Functional tests: all executors are able to read and write data to the provided storage, as allowed and via the interface; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/132) |

#### IP over Libp2p

| | |
| --- | --- |
| **Feature name** | IP over Libp2p; [feature::ip-over-libp2p](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aip-over-libp2p&first_page_size=20) |
| **Work packages** | Within the scope of [network implementation work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/network-package-implementation_technical_dependencies.html); [dependent work package in another milestone](https://nunet.gitlab.io/publisher/project-management-portal/public-alpha-solutions/work_packages/ip-over-libp2p-interface-layer_technical_dependencies.html) |
| **Code reference** | [network package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/network?ref_type=heads); [ip-over-libp2p merge request](https://gitlab.com/nunet/device-management-service/-/merge_requests/362) |
| **Description / definition of done** | [ip-over-libp2p implementation issue](https://gitlab.com/nunet/device-management-service/-/issues/398) | 
| **Timing** |  |
| **Status** | Close to completion |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Run programs and frameworks on nunet-enabled machines that need ipv4 connectivity layer |
| **Value score** | n/a |
| **Design** |  [related issues](https://gitlab.com/search?search=ip+over+libp2p&nav_source=navbar&project_id=35912922&group_id=6160918&scope=issues); [related comments](https://gitlab.com/search?group_id=6160918&project_id=35912922&scope=notes&search=ip+over+libp2p); [proof of concept implementation](https://gitlab.com/nunet/misc-experiments/ipoverlibp2p-poc-orchestration)  |
| **Impacted functionality** | Ability to integrate with third party frameworks for orchestration (e.g. Kubernetes, others) as well as run distributed software (database clusters, etc.); Will be mostly used for the [Public Alpha Solutions milestone](https://app.gitbook.com/o/HmQiiAfFnBUd24KadDsO/s/UoAe47nbLYU9lZHCUSJH/public-alpha-solutions)  |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: (1) spawn a ipv4 network for containers running on different machines to directly interact with each other; (2) Access compute providers via Kubernetes cluster / orchestrate jobs via Kubernetes cluster (advanced);  <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/131)|


#### Observability and Telemetry

| | |
| --- | --- |
| **Feature name** | Observability and Telemetry design and implementation; [feature::observability](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aobservability&first_page_size=20) |
| **Work packages** | [telemetry work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/telemetry-package-implementation_technical_dependencies.html) |
| **Code reference** | [telemetry package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/telemetry?ref_type=heads) |
| **Description / definition of done** | [telemetry interface implementation issue with full description](https://gitlab.com/nunet/device-management-service/-/issues/412) <br>[default elasticsearch collector issue with full description](https://gitlab.com/nunet/nunet-infra/-/issues/198)| 
| **Timing** |  |
| **Status** | In progress; |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** | Development team by having full visibility of code logs and traces across the testing networks; Potential integrators and use-case developers for debugging their applications running on decentralized hardware; QA team accessing test logs; |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** |  [observability interface in yellow-paper](https://app.gitbook.com/o/HmQiiAfFnBUd24KadDsO/s/nUIl2GGpV9Wq3xiFlif2/public-technical-documentation/publisher/platform-yellow-paper/main/c_observability-framework) |
| **Impacted functionality** | Logging, tracing and monitoring of decentralized computing framework on any level of granularity; Constitutes a part of developer tooling of NuNet, which will be used by both internal team as well as community contributors |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: after logging is implemented via telemetry interface and default logging is elasticsearch collector; all telemetry events are stored in elasticsearch database and can be analyzed via API / Kibana dashboard; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/129) |

#### <a id="#definition-of-compute-workflowsrecursive-jobs"></a>Definition of compute workflows/recursive jobs

| | |
| --- | --- |
| **Feature name** | Structure, types and definitions of compute workflows / recursive jobs; [feature::workflow-definition](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Aworkflow-definition&first_page_size=20) |
| **Work packages** | [jobs work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/jobs-package-implementation_technical_dependencies.html) |
| **Code reference** | [jobs package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/jobs?ref_type=heads) |
| **Description / definition of done** | [jobs design description](https://gitlab.com/nunet/architecture/-/issues/245) | 
| **Timing** |  |
| **Status** | In progress; |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Set NuNet job definition format and types in order to be able to orchestrate them via orchestration mechanism |
| **Value score** | n/a |
| **Design** | [jobs design description](https://gitlab.com/nunet/architecture/-/issues/245)  |
| **Impacted functionality** | Used in job orchestration in order to be able to search and match fitting machines that are connected to the network; Related to Capability / Comparison model |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: Ability to represent any job that can be represented via kubernetes / nomad in nunet job fomat / convert to inner type and orchestrate its execution; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/128) |


#### Job deployment and orchestration model

| | |
| --- | --- |
| **Feature name** | Job deployment and orchestration model; [feature::job-orchestration](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Ajob-orchestration&first_page_size=20) |
| **Work packages** | [orchestrator work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/orchestrator-package-implementation_technical_dependencies.html) <br>[jobs work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/jobs-package-implementation_technical_dependencies.html) |
| **Code reference** | [orchestrator package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/orchestrator?ref_type=heads) |
| **Description / definition of done** | [jobs design description](https://gitlab.com/nunet/architecture/-/issues/245) | 
| **Timing** |  |
| **Status** | In progress; |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Submit a job description and rely on NuNet platform to execute it optimally (including finding fitting machines, connecting them to subnetworks, etc.) |
| **Value score** | n/a |
| **Design** | [specification and architecture of job orchestration (issue)](https://gitlab.com/nunet/research/core-platform/-/issues/17)  |
| **Impacted functionality** | Related to all sub-packages in the dms package and defines Capability / Comparison model |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: Submit a job described in NuNet job description format, observe its deployment and execution and returning results; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/127) |


#### Hardware capability model

| | |
| --- | --- |
| **Feature name** | Capability / Comparator model; [feature::hardware-capability](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Ahardware-capability&first_page_size=20) |
| **Work packages** | Part of the [orchestrator work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/orchestrator-package-implementation_technical_dependencies.html) |
| **Code reference** | [dms.orchestrator.matching sub-package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/dms/orchestrator/matching?ref_type=heads) <br>[related issues](https://gitlab.com/search?group_id=6160918&project_id=35912922&repository_ref=develop&scope=issues&search=capability)|
| **Description / definition of done** |  | 
| **Timing** |  |
| **Status** | Close to completion |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** | Define requested compute Capabilities (included in job definition) and machine Capabilities for searching and matching fitting machines in the network |
| **Value score** | n/a |
| **Design** | [comment on job orchestration proposal](https://gitlab.com/nunet/open-api/platform-data-model/-/merge_requests/35#note_1928384731)  |
| **Impacted functionality** | (1) Ability to match compute requirements with available Capabilities of machines, considering not only hard (hardware, etc), but also soft requirements (price, time, etc preferences from both sides of matching process); (2) Comparing different machine Capabilities (selecting best machine for a job); (3) Adding / subtracting Capabilities in order to be able to calculate machine usage in real time; See also mentions of Capability model in other functionality descriptions in this document |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: via job orchestration integration tests; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/126) |


#### Supervision model

| | |
| --- | --- |
| **Feature name** | Supervision model; [feature::supervision-model](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Asupervision-model&first_page_size=20) |
| **Work packages** | Part of the [orchestrator work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/orchestrator-package-implementation_technical_dependencies.html) |
| **Code reference** | [issue for tracking the implementation](https://gitlab.com/nunet/device-management-service/-/issues/475) |
| **Description / definition of done** |  | 
| **Timing** |  |
| **Status** | Not started |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** | [research blog on Supervision model proposal](https://nunet.gitlab.io/research/blog/posts/supervisor-model/) |
| **Impacted functionality** | Ability to build a 'decentralized' control plane on NuNet; error propagation between Actors participating in the same compute workflow; heartbeat and health-check functionalities; conceptually, supervisor model enables failure recovery and fault tolerance features in the network; related to 'remote procedure calls' functionality; |
| **Acceptance tests** | Unit tests; Functional tests / integration tests: build hierarchies of actors (nodes and allocations) that can observe each other; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/125)|

#### Tokenomics interface

| | |
| --- | --- |
| **Feature name** | Tokenomics interface; [feature::tokenomics](https://gitlab.com/groups/nunet/-/issues/?sort=created_date&state=all&label_name%5B%5D=feature%3A%3Atokenomics&first_page_size=20) |
| **Work packages** | [tokenomics work package](https://nunet.gitlab.io/publisher/project-management-portal/device-management-service-version-0-5-x/work_packages/tokenomics-package-implementation_technical_dependencies.html) |
| **Code reference** | [tokenomics package code](https://gitlab.com/nunet/device-management-service/-/tree/develop/tokenomics?ref_type=heads) |
| **Description / definition of done** | Minimal implementation of the interface in order to be able to implement micro-payments layer in the next milestone | 
| **Timing** |  |
| **Status** | In progress |
| **Team** | |
| **Strategic alignment** |  |
| **Who it benefits** |  |
| **User challenge** |  |
| **Value score** | n/a |
| **Design** | [tokenomics design work package description](https://gitlab.com/nunet/architecture/-/issues/251); <br>[tokenomics implementation work package description](https://gitlab.com/nunet/architecture/-/issues/250) |
| **Impacted functionality** | Ability to conclude peer to peer contracts between machines requesting a job and machines accepting job execution (eventually); Ability to include explicit contract information into each job invocation request, independently of the type of contract and micro-payment channels  implementation |
| **Acceptance tests** | Unit tests; Functional tests / integration tests as part of job orchestration; <br>[tracking issue](https://gitlab.com/nunet/test-suite/-/issues/124)|


