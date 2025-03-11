PROTO_DIR := proto/v1/common
GO_OUT_DIR := proto/generated/v1/common

PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
PROTOC := protoc

UNAME=$(shell uname)
ARCH=$(shell uname -m)

NO_COMMITS=$(shell git rev-list $(shell git describe --tags --always --abbrev=0)..HEAD --count )

DMS_VERSION=$(shell git describe --tags --always --abbrev=0 --dirty)-$(NO_COMMITS)-$(shell git rev-parse --short HEAD)
GO_VERSION := $(shell go version | awk '{print $$3}' | sed 's/go//')
BUILD_DATE := $(shell date -Iseconds)
BUILD_HASH := $(shell git rev-parse HEAD)

LDFLAGS := \
	"-X 'gitlab.com/nunet/device-management-service/cmd.Version=$(DMS_VERSION)' \
	-X 'gitlab.com/nunet/device-management-service/cmd.GoVersion=$(GO_VERSION)' \
	-X 'gitlab.com/nunet/device-management-service/cmd.BuildDate=$(BUILD_DATE)' \
	-X 'gitlab.com/nunet/device-management-service/cmd.Commit=$(BUILD_HASH)'"

GOFLAGS := "-buildvcs=false"

.PHONY: all clean linux_amd64 darwin_arm64 license mock

all:
	@if [ $(UNAME) = Linux ]; then\
		make linux_amd64;\
	elif [ $(UNAME) = Darwin ]; then\
		make darwin_arm64;\
	fi

linux_amd64:
	@echo "Building for Linux AMD64..."
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o builds/dms_linux_amd64 -ldflags=$(LDFLAGS) .

linux_arm64:
	@echo "Building for Linux ARM64..."
	go mod tidy
	@if [ $(ARCH) = "aarch64" ]; then\
		echo "Building ON ARM64...";\
		GOOS=linux GOARCH=arm64 go build -o builds/dms_linux_arm64 -ldflags=$(LDFLAGS) .;\
	elif command -v aarch64-linux-gnu-gcc > /dev/null 2>&1; then\
		echo "Cross Compiling for aarch64...";\
		CGO_ENABLED=1 CC_FOR_TARGET=gcc-aarch64-linux-gnu CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build -o builds/dms_linux_arm64 -ldflags=$(LDFLAGS) .;\
	else\
		echo "arm64 - no compiler found";\
	fi

linux_arm32:
	go mod tidy
	@if [ $(ARCH) = "armv7l" ] || [ $(ARCH) = "armv6l" ]; then\
		echo "Building ON ARM32...";\
		GOOS=linux GOARCH=arm64 go build -o builds/dms_linux_arm64 -ldflags=$(LDFLAGS) .;\
	elif command -v arm-linux-gnueabihf-gcc > /dev/null 2>&1; then\
		echo "Cross Compiling for arm32...";\
		CGO_ENABLED=1 CC_FOR_TARGET=gcc-arm-linux-gnueabihf CC=arm-linux-gnueabihf-gcc GOOS=linux GOARCH=arm go build -o builds/dms_linux_arm32 -ldflags=$(LDFLAGS) .;\
	else\
		echo "arm32 - no compiler found";\
	fi

darwin_arm64:
	@echo "Building for Darwin ARM64..."
	go mod tidy
	GOOS=darwin GOARCH=arm64 go build -o builds/dms_darwin_arm64 -ldflags=$(LDFLAGS) .

darwin_amd64:
	@echo "Building for Darwin AMD64..."
	go mod tidy
	GOOS=darwin GOARCH=amd64 go build -o builds/dms_darwin_amd64 -ldflags=$(LDFLAGS) .

lint:
	golangci-lint run --max-issues-per-linter=200

clean:
	@echo "Cleaning up..."
	rm -rf builds/

setcap: 
	sudo setcap cap_net_admin,cap_sys_admin+ep ./test/integration/dms

itest:
	@echo "Running integration tests"
	go build -o ./test/integration/dms -ldflags=$(LDFLAGS)
	make setcap
	go test -v ./test/integration/... -tags=integration -timeout=5m

generate:
	$(PROTOC) --proto_path=$(PROTO_DIR) --go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative $(PROTO_FILES) --go_opt=Mcommon.proto=proto/generated/common

LICENSE_FLAGS := -v \
		-l="apache" \
		-f copyright.txt \
		-c "NuNet" \
		-ignore "**/*.md" \
		-ignore "**/*.html" \
		-ignore "**/*.css" \
		-ignore "**/*.scss" \
		-ignore "**/*.yml" \
		-ignore "**/*.yaml" \
		-ignore "**/*.js" \
		-ignore "**/*.sh" \
		-ignore "*Dockerfile"

license:
	@echo "  →→  \033[1;36m$(if $(CHECK),Checking,Adding) license headers...\033[0m"
	go install github.com/google/addlicense@v1.1.1
	addlicense $(LICENSE_FLAGS) $(if $(CHECK),-check) .

arch=$(shell uname -m)
FC_TEST_DATA_PATH = ./executor/firecracker/testdata

testdata_objects = \
$(FC_TEST_DATA_PATH)/rootfs.ext4 \
$(FC_TEST_DATA_PATH)/vmlinux.bin

testdata: $(testdata_objects)
	@echo "Preparing test data..."

$(FC_TEST_DATA_PATH)/rootfs.ext4:
	@echo "Downloading rootfs.ext4..."
	mkdir -p $(FC_TEST_DATA_PATH)
	curl -L -o $@ https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4

$(FC_TEST_DATA_PATH)/vmlinux.bin:
	@echo "Downloading vmlinux.bin..."
	mkdir -p $(FC_TEST_DATA_PATH)
	curl -L -o $@ https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/$(arch)/kernels/vmlinux.bin

mock:
	@echo "Generating mock files... :("
	# API mocks
	mockgen -destination=api/mock_network_test.go -source=network/network.go -package=api -exclude_interfaces=Messenger
	
	# CMD mocks
	mockgen -destination=cmd/mock_gpu_manager_test.go -source=types/hardware.go -package=cmd -exclude_interfaces=HardwareManager,GPUConnector
	mockgen -destination=cmd/mock_docker_client_test.go -source=executor/docker/client.go -package=cmd
	
	# DMS/Onboarding mocks
	mockgen -source=db/repositories/generic_entity_repository.go -destination=dms/onboarding/mock_generic_entity_repository_test.go -package=onboarding
	mockgen -destination=dms/onboarding/mock_hardware_manager_test.go -source=types/hardware.go -package=onboarding
	mockgen -destination=dms/onboarding/mock_resource_manager_test.go -source=types/resources.go -package=onboarding -exclude_interfaces=UsageMonitor,ResourceOps
	
	# DMS/Hardware mocks
	mockgen -destination=dms/hardware/gpu/mock_gpu_connector_test.go -source=types/hardware.go -package=gpu -exclude_interfaces=HardwareManager,GPUManager
	mockgen -destination=dms/hardware/mock_gpu_connector_test.go -source=types/hardware.go -package=hardware -exclude_interfaces=HardwareManager,GPUConnector
	
	# DMS/Node mocks
	mockgen -destination=dms/node/mock_geoip_locator_test.go -source=types/types.go -package=node
	mockgen -destination=dms/node/mock_network_test.go -source=network/network.go -package=node -exclude_interfaces=Messenger
	mockgen -destination=dms/node/mock_orch_registry_test.go -source=dms/orchestrator/registry.go -package=node
	mockgen -destination=dms/node/mock_orchestrator_test.go -source=dms/orchestrator/orchestrator.go -package=node
	mockgen -destination=dms/node/mock_executor_test.go -source=types/executor.go -package=node
	mockgen -destination=dms/node/mock_actor_test.go -source=actor/interface.go -package=node
	mockgen -destination=dms/node/mock_hardware_manager_test.go -source=types/hardware.go -package=node
	mockgen -destination=dms/node/mock_resource_manager_test.go -source=types/resources.go -package=node -exclude_interfaces=UsageMonitor,ResourceOps
	mockgen -destination=dms/node/mock_allocator_test.go -source=dms/node/allocator.go -package=node
	
	# DMS/Resources mocks
	mockgen -source=db/repositories/generic_repository.go -destination=dms/resources/mock_generic_repository_test.go -package=resources
	mockgen -source=db/repositories/generic_entity_repository.go -destination=dms/resources/mock_generic_entity_repository_test.go -package=resources
	mockgen -destination=dms/resources/mock_hardware_manager_test.go -source=types/hardware.go -package=resources
	
	# DMS/Jobs mocks
	mockgen -destination=dms/jobs/mock_network_test.go -source=network/network.go -package=jobs -exclude_interfaces=Messenger
	mockgen -destination=dms/jobs/mock_executor_test.go -source=types/executor.go -package=jobs
	mockgen -destination=dms/jobs/mock_actor_test.go -source=actor/interface.go -package=jobs
	mockgen -destination=dms/jobs/mock_resource_manager_test.go -source=types/resources.go -package=jobs -exclude_interfaces=UsageMonitor,ResourceOps

	# DMS/Orchestrator mocks
	mockgen -destination=dms/orchestrator/mock_network_test.go -source=network/network.go -package=orchestrator -exclude_interfaces=Messenger
	mockgen -destination=dms/orchestrator/mock_actor_test.go -source=actor/interface.go -package=orchestrator
