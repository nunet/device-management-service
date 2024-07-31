<!-- New changes go on top. But below these comments. -->

<!-- We can track changes from Git logs, so why we needs this file? 

Guiding Principles
- Changelogs are for humans, not machines.
- There should be an entry for every single version.
- The same types of changes should be grouped.
- Versions and sections should be linkable.
- The latest version comes first.
- The release date of each version is displayed.
- Mention whether you follow Semantic Versioning.

Types of changes
- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Removed` for now removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

-->

## [0.4.174](#481)

### Deprecated
- deprecated logbin

## [0.4.173](#489)

### Changed
- changed Basic Controller and its tests accordingly to DB repositories for StorageVolumes
- minor change on pkg to align with changes on VolController test suite
- improved error handling in clover generic repo methods

## [0.4.172](#502)

### Fixed
- unavailable log stream implementation for firecracker executor

## [0.4.171](#473)

### Added
- Actor Module (#473)
- Network Module (#445)

## [0.4.170](#386)

### Added
- Support for Clover DB

## [0.4.169](#428)

### Added
- Volume Controller

## [0.4.168](!349)

### Deprecated
- deleted the following packages
    - docker
    - firecracker
    - integrations/oracle
    - internal/klogger
    - internal/messaging
    - internal/tracing
    - libp2p
### Changed
- moved /internal/logger into /telemetry
- moved /docs into /api/docs
- moved integration/tokenomics into /utils/blockchain.go and /cardano into /utils/cardano

## [0.4.167](#409)

### Changed
- Docker executor contianer run pulls image if it doesn't exist locally
- Readme update on testing

### Added
- Executor tests and test data fetch using make

## [0.4.166](#396)

### Added
- Firecracker executor

## [0.4.165](#346)

### Added
- Generic executor package interface and its docker implementation

## [0.4.164](#340)

### Fixed
- Tests that cause timeout or crash

### Removed
- Spans from p2p messaging due to interruption of tests

### Changed
- Moved peer filter call out of discover

## [0.4.163](#364)

### Fixed
- Total vram instead of visible only for AMD GPUs

## [0.4.162](#206)

### Added
- Repositories for db module. First phase of db refactoring 

## [0.4.161](#353)

### Changed
- Moved onboarding and resource related code to dms package
- Refactored CheckOnboarding

## [0.4.160](#366)

### Fix
- Problem with vm start request on firecracker depreq

## [0.4.159](#376)

### Removed
- TxHash verification on incoming deployment request

## [0.4.158](#361)

### Changed
- Reorganized REST api functions to an api package
- Separate REST handler and actual functionality

## [0.4.157](#345)

### Fixed
- Capacity and info output format
- Log and offboard command crash

### Added
- Allow setting default depreq peer with main app cli
- CLI auto-completion configured for bash and zsh on postinst

### Removed
- Deprecated bash CLI and replaced with main app cli
- Deleted cli_test for bash CLI

## [0.4.156](#337)

### Fixed
- Improper state from existing metadatafile but no private key
## [0.4.155](#333)

### Added
- Background tasks scheduler package

## [0.4.154](nunet/research/network-tokenomics/public-alpha-model#8)

### Added
- Allow user to set price in NTX for onboarded compute resource

## [0.4.153](#257)

### Added
- Device cmd for device status management
- Device job-availability set during onboarding

## [0.4.152](#321)

### Changed
- Removed unnecessary fs argument in db connection
- Updated tests
- Avoid panic on unavailable telemetry collector

## [0.4.151](#315)

### Changed
- Update commmand run functions to return errors
- Organize command implementations and their backends
- Unit test for each command

## [0.4.150](#327)

### Fixed
- Reset outbound depreq stream when other side closes

## [0.4.149](#324)

### Changed
- add endpoint to list available checkpoint files for resuming
- rename a few handlers to align with others
- change ownership of checkpoint files to assumed user in container

## [0.4.148](#298)

### Changed
- Use progress checkpoint file to request a resume job

## [0.4.147](#323)

### Added
- Created REST endpoint for updating transaction status in db
- Blockchain util function to get list of utxos of smart contract

## [0.4.146](#328)

### Changed
- Allow targeting of peer coming from the DeploymentRequest payload

## [0.4.145](#297)

### Changed
- Correct error propagation in libp2p, docker and onboarding packages

## [0.4.144](#327)

### Fixed
- close the libp2p stream when CP sends a success: false

## [0.4.143](oracle#24)

### Changed
- Fixed unique constraint error while save service into SP's DMS.
- Update transaction status to running.

## [0.4.142](#295)

### Added
- Keep a record of images pulled by DMS in the database to track dangling images
- Utility functions in the docker package for search, remove and get containers using an image.
- Added a daemon function to cleanup dangling images

## [0.4.141](#166)

### Added
- Temporary setter for DMSp2p
- Routes tests

### Changed
- Allow db to be initiated on a mock fs

### Fixed
- Fix /transactions responses when tx don't exist
- Fixed utils/network tests


## [0.4.140](#292)

### Added
- TxHash validation by CP on depReq receive before running job

## [0.4.139](#292)

### Added
- Function `StopAndRemoveContainer` in the docker package
- Logger for the dms package
- Function `SanityCheck` that checks and kills lingering containers and correct available resources on dms start
## [0.4.138](#292)

### Added
- Command gpu and sub-commands status, capacity and onboard
- Port onboard-ml and offboard commands from bash script
- Add scripts for driver and container runtime installation inside maint-scripts

### Changed
- Update InternalAPIURL and MakeInternalRequest utility functions to support queries
- Move String method inside gpudetect package

## [0.4.137](#309)

### Added
- Added wallet addresses in request-reward endpoint to return dashboards
- Added parameters of  in request-reward endpoint to return dashboards
- Calculated time execution of job container

### Changed
- Changed the transactions endpoints to filter them on dashboards
## [0.4.136](#283)

### Added
- Oracle mock to enable testing
- Refactored HandleRequestReward function
- Created Unit Tests & Mocks for routes
## [0.4.135](#308)

### Added
- Wallet address self validation on startup
- CP wallet address validation before job deployment request

## [0.4.134](#126)

### Added
- add timeout for container, use parameter from dep req
- handle new fields from SPD regarding resuming
- save tar.gz of /workspace directory in container every logUpdateInterval
- add volume binding before starting a resuming container (extracted tar.gz -> /workspace)
- add event handler at timeout to archive & send the final progress tarball
- send sha256 checksum along with tarball to SPD

## [0.4.133](#314)

### Fixed
- GPU detection on job start (#314)
## [0.4.132](#273)

### Changed
- Support Darwin arm64 and amd64 builds
- Makefile for building
- Separate GPU detection for linux/darwin , arm64/amd64
- No GPU detection support for Darwin
## [0.4.131](#249)

### Changed
- Changed the telemetry/ folder from firecracker/ to the repository's root.
- Created some functions to abstract the additions/subtractions of resource usage and available resources.
- Unit/Integration tests for most of the functions of Telemetry package, the main one being calcFreeResources()
## [0.4.130](#9)

### Added
- Allow setting device status (online/offline)
- Rest endpoints and CLI commands to set device status
## [0.4.129](#278)

### Added
- Create WebSocket client for handling communication with server
- Add start chat, join chat, clear chat and list chat commands
- Make openStream struct exportable and reuse it for listing chats
- Fix typo in welcome message when starting a chat
- Add libsystemd-dev as build/dev dependency
## [0.4.128](#220)

### Added
- Port the commands to Go: peer list, peer self, onboard, info, capacity, wallet new, resource-config, log
- Change pattern of commands to APP VERB NOUN
- Add --dht flag for list peers command
- Format output of show capacity command in YAML manner (suitable to change)

## [0.4.127](#296)

### Added
- File transfer between peers (#147 -> #196)
## [0.4.126](nunet/ml-on-gpu/ml-on-gpu-webapp/-/issues/41#note_1490827796#41)

### Changed 
- Request-reward endpoint to return the new parameters from oracle.
- Request-service endpoint to return the hashes from oracle.
### Deprecated
- `signature` and `oracle_message` parameters.
### Added 
- oracle paramters for `withdrawReq` and `fundingReq`

# [0.4.125.1](#306)
### Fixed
- increase the threshold to use all onboarded resources
- prohibit saying no enough resources
# [0.4.125](#287)
### Fixed
- Filter correct job/services executed on machine with tx hash for request reward
# [0.4.124](!212)
### Fixed
- Fixed a bug where the NVIDIA Container Runtime Installation would be skipped when onboarding on mining operating systems with NVIDIA GPUs
- Fixed showing the AMD ROCm Kernel Driver detection
- Fixed a bug where DMS tries to deploy NVIDIA GPU container after pulling AMD GPU image

### Changed
- Revised container runtime installation script and condition for adding the render group for AMD GPUs if it exists (Ubuntu > 18.04)
- Set Service Names based on imageName to create distinct services for every image
# [0.4.123](#284)
### Fixed
- Update timestamp when call service events are changed
- Keep elastic tokens for different channels
# [0.4.122](#282)
### Fixed
- Correct new elastic parameters on new token
- Correctly save new keys for peer info on subsequent creation
# [0.4.121](#281)
### Fixed
- Correct nodeid on processUsage telemetry
# [0.4.120](!221)
### Fixed
- Added validation on Onboard API for dedicated capacity to NuNet
- Removed redundancy of binding request data to JSON
- Fix heartbeat invalid token issue
- Update protocol versions due to depreq backward incompatibility
# [0.4.119](#263)
### Removed
- Removed RequestTracker table from DMS.
- Removed stats_db related code from Onboard API
# [0.4.118](#214)
### Added
- Logger class with the basic log levels
- Partial/main functionalities logging to elk

### Removed
- Removed telemetry spans
# [0.4.117](#267)
### Added
- Endpoint to retrieve list of tx hashes for jobs done on machine

### Changed
- Accept tx hash from spd on send-status
- Include tx hash in depreq to CP
- Use tx hash to match job done during claim
# [0.4.116](#255)
### Changed
- Replace gist with logbin for log storage

# [0.4.115](#247)
### Added
- Send telemetry events to ELK stack

### Removed
- Deprecated StatsDB

# [0.4.114](oracle#4)
### Changed
- New oracle addresses
- TLS for oracle RPC
# [0.4.113](#209)
### Changed
- Start moving CLI from bash script to main app
# [0.4.112](#200)
### Changed
- Allow comments in config file
# [0.4.111](#242)
### Added
- `Offboard` endpoint and CLI command
- Onboarding status endpoint
# [0.4.110](#241)
### Added
- Wallet address validation
# [0.4.109](#256)
### Added
- Add request and response struct for swagger docs
- Generate latest docs
- Add swag directly to project
# [0.4.108](#243)
### Fixed
- Native ping with cancel
- DHT cleanup routine context handling
- Reset ping stream whe not replying
# [0.4.107](#252)
### Fixed
- Remove wrong context implementation on FetchKadDHTContents
# [0.4.106](#243)
### Fixed
- Refactor discovery and DHT update routines
- Set deadlines for fetching DHT data and pinging a peer

# [0.4.105](#196)
### Fixed
- Add verification query when onboarding an already onboarded machine (fix for #196)

# [0.4.104](#195)
### Fixed
- Ignore decimal values for resource amount on onboard and resource-config (fix for #195)

# [0.4.103](!191)
### Changed
- Added Binds in AMD GPU Host Config to autocreate device file

# [0.4.102](!186)
### Changed
- Improve GPU related logs

# [0.4.101](#230)
### Fixed
- ML Onboarding on Windows Subsystem for Linux (WSL)
- Handle missing NVIDIA/AMD GPU(s) separately on Linux
- Reboot recommendation after GPU onboarding

# [0.4.100](#227)
### Fixed
- Error handling on heartbeat module

# [0.4.99](!183)

### Fixed
- A scenario where machines without AMD GPUs wouldn't allow NVIDIA GPUs to be monitored
# [0.4.98](#239)

### Removed
- Old DHT update implementation

### Changed
- Better DHT fetching efficiency
# [0.4.97](#221)

### Added
- Fixed a scenario where machines with AMD GPUs wouldn't be detected if NVIDIA GPUs were already present
- Modify systemd PATH if mining os detected on onboard-gpu
- New service package as replacement for docker package (wip: refactoring #204)
# [0.4.96](#238)

### Changed
- Temporary patch for job deployment problem from kad-dht fetch taking too long
# [0.4.95](!179)

### Changed
- Print error message when bootstrap node couldn't connect

### Fixed
- Set total and free vram values for AMD gpus
# [0.4.94](#218)

### Changed
- Websocket connection handling to global
- Remove depreciated DHT update after container start
- Deployment update/response on jobstatus=running to webapp

# [0.4.93](#202)

### Changed
- Detect the GPU with the highest free VRAM, to decide whether to deploy an NVIDIA or AMD GPU container
- Separate images for AMD and Nvidia GPUs
# [0.4.92](#210)

### Changed
- GPU detection and info for both NVIDIA and AMD GPUs
- GPU model fields all strings
- Use NVIDIA/go-nvml for gpu info and jaypipes/ghw detection
# [0.4.91](#216)

### Changed
- Recovery on unforeseen errors in heartbeat module
- No heartbeat when machine not onboarded / no libp2p host
- Use internal logger for heartbeat module
# [0.4.90](!159)

### Changed
- Improved and optimized GPU onboarding for mining operating systems
- Improvement on `nunet onboard-ml` with revamp of the entire functionality

### Added
- Support for AMD GPU
- Detection of AMD ROCm and HIP with `nunet capacity --rocm-hip`

# [0.4.89](#213)

### Changed
- Get available resources from metadata file

# [0.4.88](#178)

### Added
- Instrument libp2p communication with ELK
- Heartbeat every minute
# [0.4.87](#162, !160)

### Changed
- Stored peerInfo of nodes in Kad-DHT
- Added a new endpoint to list Kad-DHT contents
- Fixed a bug with free resource calculation after job deployment
# [0.4.86](#182)

### Changed
- Allow setting a default peer for sending deployment requests to
- Do not panic on improperly formatted deployment update
- Bug fixes on job completion and second time deployment request
# [0.4.85](#174)

### Changed
- Send logs from CP-DMS to SP-DMS in chunks of 2 minutes (configurable).
- Job status messages "job-failed" and "job-complete" expected by SP-DMS
# [0.4.84](#201)

### Changed
- Refuse replying to depReq ping if already running a job
# [0.4.83](stats-database#17)

### Added
- Time elapsed and average network bandwidth used during a job in Service_call event
# [0.4.82](#181)

### Added
- Man page that helps with nunet cli
# [0.4.81](#190)

### Changed
- Fix for #190 by improving how depReqFlat is updated during deployment requests
- Return job status to SP webapp when job finishes on compute provider
- Fix for #163 by downgrading kad-dht package to 0.22.0 

# [0.4.80](!146)

### Changed
- Fix CLI unit tests
- Allow SetConfig to set parameters at runtime
- Improve UUID generation and db storage
# [0.4.79](#185)

### Added
- Remove /etc/nunet during dpkg purge
# [0.4.78](#180)

### Changed
- Replace ubuntu-drivers with nvidia meta packages for better compatibility
# [0.4.77](#179)

### Added
- Add config module for configuring DMS runtime

# [0.4.76](#171)

### Changed
- Fixes #171 parse error on `nunet peer list`
# [0.4.75](#176)

### Added
- Respond with {"action": "job-submitted"} just before deployment request is handed over to compute provider
# [0.4.74](!135)

### Changed
- Generate Events whenever there is an update to JobStatus field in service table

### Changed
- Send events on the same stream which is responsible for deployment response
- Update DeploymentRequestFlat table with JobStatus on service provider side
- Raise an error on frontend if a job is already running
- Close libp2p network stream when job is done either with success or failure
# [0.4.73](#175)

### Changed
- Increase timeout when connecting to Oracle
# [0.4.72](#173)

### Changed
- Bug fix: disable local address filter when not on server mode
## [0.4.71](!137)

### Changed
- No error on calcUsedResources when no services

## [0.4.70](#170)

### Changed
- Avoid error on unreplied ping
- Fix `dial to self attempted` bug
- ntx_payment event on claim by CP instead of on depReq by SP
- Avoid panic on deployment errors and send appropriate deployment response
- Avoid panic on gist update and dht update errors

### Added
- Oracle instances for channels other than team channel
## [0.4.69](#168)

### Changed
- Obtain gist token from a public endpoint
- Send depResp and close stream if unable to create gist
## [0.4.68](!129)

### Changed
- Raise HTTP exceptions for internal DMS error occurred on route handlers
## [0.4.67](#165)

### Changed
- Added ServiceCall event back after removed by mistake
## [0.4.66](#167)

### Changed
- Refactored statsdb device resource change calls
## [0.4.65](#164)

### Changed
- Avoid fetching peers at multiple places
- Increase relay limits and reservation resources
- Use only NuNet peers for relay and DHT update 

### Removed
- Removed the libp2p bootstrap nodes
## [0.4.64](#157)

### Added
- Rest endpoint to change amount of resource onboarded to NuNet
- Subcommand `resource-config`
- StatsDB event call on onboarded resource amount change
## [0.4.63](#161)

### Changed
- Make sure deviceinfo is set for outlier machines such as VMs (#161 bug fix)
## [0.4.62](#149)

### Changed
- Enabled autorelay and holepunching
- Upgraded libp2p and golang versions to 0.27 and 1.20
## [0.4.61](#109)

### Changed
- Re-establish connection to peers on network change
- Ping peers before sending Deployment requests
- Added functions to set up PubSub communication
## [0.4.60](#134)

### Added
- implemented ntx_payment event for sending transaction info to stats database

## [0.4.59](!114)

### Changed
- Delete entries in services table after response with oracle
## [0.4.58](#128)

### Changed
- Machine uuid for identification before onboarding and peerID attribute for opentelemetry
- Fix logger configuration for opentelemetry
- Remove unncessary env check for debug print and minor log format fix

## [0.4.57](!113)

### Changed
- Fix for base64 issue (cause for gist not being received by webapp)
- Improve log formatting
## [0.4.56](!112)

### Changed
- Debug mode DHT update interval env var and logging fix
## [0.4.55](!111)

### Changed
- Decode base64 message coming from libp2p stream
## [0.4.54](#153)

### Changed
- Set peerInfo struct to empty instead of removing peer from peerstore
## [0.4.53](#154)

### Changed
- Mark all records as deleted when forwarding deployment request to compute provider.
- Read first non-deleted record when reading temporary record.
## [0.4.52](#150)

### Changed
- Added List DHT peers endpoint
- Remove old peers from DHT
## [0.4.51](!106)

### Changed
- Set JobStatus to be finished with errors if container exited with non-zero exit status.
- Set JobStatus to be finished without errors if container exited with zero exit status.
- Return 102 status code when container is still running. DMS won't contact Oracle in such case.
## [0.4.50](!104)

### Changed
- add dht/peers route and debug print on deployment request
- bug fixes and debug prints with debug env var
## [0.4.49](!102)

### Changed
- filter machines for cpu only ml jobs
- fence index error
## [0.4.48](#143)

### Changed
- Fix for problem with CPU-only job deployment
## [0.4.47](#142)

### Changed
- Save LogURL and other service related data on service run
- Fetch LogURL and other service related data on request reward
## [0.4.46](!96)

### Changed
- Correct the wrong formatting of deployment response.
- Minor refactoring
## [0.4.45](#144)

### Changed
- Quick install script to query for blockchain when creating wallet
- Fix instruction on readme to avoid running in a subshell
- Use NuNet bootstrap servers and default libp2p host options 
## [0.4.44](!94)

### Changed
- Fix for the problem with non-flat model db migration
## [0.4.43](#123)

### Changed
- DMS integration with Oracle to get blockchain data
  
## [0.4.42](#139)

### Changed
- Replaced GPU detection library 
  
## [0.4.41](#125)

### Added
- The logic to handel the choice of CPU and GPU

## [0.4.40](#137)

### Added
- Reduced DHT update interval

### Removed
- Calls to StatsDB in GPU deployment. Refer to #138

## [0.4.39](#124)

### Added
- Included pre-commit hook and a dev-setup.sh.
### Changed
- Fix for CORS error.
## [0.4.38](#133)

### Added
- Use public key from DB instead of attempting to extract from ID
## [0.4.37](#105)

### Added
- NuNet Hosted Bootstrap Servers
## [0.4.36](#135)

### Added
- Server mode to disable local network scan on datacenters
## [0.4.35](#132)

### Added
- Detect GPU while onboarding and update GPU flag on DHT

## [0.4.34](#115)

### Added
- Instrumentation
## [0.4.33](#119)

### Removed
- Removed all NuNet Adapter usage and installation
## [0.4.32](#129)

### Added
- Sending DHT updates periodically
## [0.4.31](#107)

### Added
- Deployment request with libp2p messaging
- Simple chat between peers using CLI
## [0.4.30](#106)

### Added
- Implemented DHT on libp2p with similar schema to the nunet adapter
- Helper functions to query the libp2p DHT

## [0.4.29](nunet/ml-on-gpu/ml-on-gpu-service#14)
### Changed
- Updated nunet info command to detect GPUs
- Updated nunet onboard-ml to detect ML images after the command is run
### Added
- New nunet available --gpu-status command to show GPUs metrics in real-time
- New nunet available --cuda-tensor command to check availability of CUDA and Tensor Cores

## [0.4.28](#104)

### Changed
- Improvement on discoverability
- Increase deadline on getSelfNodeID 
- Remove unnecessary code
### Added
- Missing Dependencies on Deb Package

## [0.4.27](#103)

### Added
- Libp2p node 
-  `nunet peer list` prints peers found by the libp2p node 

## [0.4.26](#92)
### Changed

Send the deployment response container log gist URL to GPU ML user.

## [0.4.25](#111)

### Added
- Unit tests for CLI

## [0.4.24](#85)

### Added
- Unit tests for /onboarding REST endpoints

## [0.4.23](#90)

### Changed
- Separated StatsDB GRPC address for different onboarding channels

## [0.4.22](#81)

### Added
- Enable wallet creation for the Cardano blockchain

## [0.4.21](#93)

### Changed
- Enable GPU onboarding for WSL users

## [0.4.20](#95)

### Changed
- Refactor logger

## [0.4.19](#75)

### Changed
- Remove random amount of wait in `Onboard` handler.
- Move telemetry code to `InstallRunAdapter` for faster request-response cycle.

## [0.4.18](#89)

### Fixed
- Fixed VM network config

## [0.4.17](documentation#19)

### Added
- CLI command to collect log and return path

## [0.4.16](#80)

### Added
- CLI command to collect log and return path

## [0.4.15](#63)

### Added
- Implementation of deployement request status information propagation to statsdb for ML job deployment.

## [0.4.14](#78)

### Added
- New gRPC service for retreiving Master Community's public key

## [0.4.13](#60)

### Added
- New grpc service for incoming messages and message receiver 
- Preliminary deployment request handler for cardano and gpu usecases

## [0.4.12](#69)

### Added
- shell command for websocat 
- websocat installation

## [0.4.11](#49)

### Added
- Passing fields required for Remote Shell authorization on VMs.

## [0.4.10](!41)

### Added
- Ping-pong websocket implementation. This connection will be used to send commands and receive output from remote shell.

## [0.4.9](#46)

### Added
- Calculate telemetry of docker containers and update DHT.
- Missing dependency `bc` in DEBIAN/control

### Changed 
- Moved DHT update grpc call to run in a separate thread.
- Image used in ci for building in order to support at least glibc-2.27

## [0.4.8](#73)

### Added
- Handle onboarding channels for separate networks corresponding to https://gitlab.com/nunet/nunet-adapter/-/issues/108
- Workaround for #74

## [0.4.7](#64)

### Added
- Send docker logs for running container to GitHub's Gist

## [0.4.6](#63)

### Added
- Added stats_db grpc calls.

## [0.4.5](#67)

### Added
- Added a grpc-client to access cardano-cli
- Added a function interface to run cardano-cli commands
- Removed unused functions

## [0.4.4](#61)

### Added
- Added command for installing NVIDIA GPU driver and Container Runtime in nunet CLI.
- Added command for pre-downloading ML docker images in nunet CLI. 

## [0.4.3](#66)

### Fixed
- Fix 500 response on `/vm/*` endpoints.

### Removed
- Remove internal endpoints.

### Changed
- Refactor `vm` module for speed and readability.
- the websocat installation to include $archdir

## [0.4.2](#57)

### Added
- Trigger docker deployment 
- the shell command which takes node id 

### Removed
- Don't remove image after GPU load is run.

### Fixed
- Fix one logical error where DMS on compute provider sent error ack for expired message. 

