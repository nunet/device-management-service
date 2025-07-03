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

.PHONY: all clean linux_amd64 darwin_arm64 license

all:
	@if [ $(UNAME) = Linux ]; then\
		if [ $(ARCH) = "aarch64" ]; then\
			make linux_arm64;\
		elif [ $(ARCH) = "armv6l" ]; then\
			make linux_arm32_v6l;\
		elif [ $(ARCH) = "armv7l" ]; then\
			make linux_arm32_v7l;\
		elif [ $(ARCH) = "x86_64" ]; then\
			make linux_amd64;\
		else\
			echo "Unsupported architecture: $(ARCH)";\
			exit 1;\
		fi;\
	elif [ $(UNAME) = Darwin ]; then\
		if [ $(ARCH) = "arm64" ]; then\
			make darwin_arm64;\
		elif [ $(ARCH) = "x86_64" ]; then\
			make darwin_amd64;\
		else\
			echo "Unsupported architecture: $(ARCH)";\
			exit 1;\
		fi;\
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

linux_arm32_v6l:
	go mod tidy
	@if [ $(ARCH) = "armv6l" ]; then\
		echo "Building ON armv6l...";\
		GOOS=linux GOARCH=arm GOARM=6 go build -o builds/dms_linux_arm32_v6l -ldflags=$(LDFLAGS) .;\
	elif command -v arm-linux-gnueabihf-gcc > /dev/null 2>&1; then\
		echo "Cross Compiling for armv6l...";\
		CGO_ENABLED=1 CC_FOR_TARGET=gcc-arm-linux-gnueabihf CC=arm-linux-gnueabihf-gcc GOOS=linux GOARCH=arm GOARM=6 go build -o builds/dms_linux_arm32_v6l -ldflags=$(LDFLAGS) .;\
	else\
		echo "arm-linux-gnueabihf - no compiler found";\
	fi

linux_arm32_v7l:
	go mod tidy
	@if [ $(ARCH) = "armv7l" ]; then\
		echo "Building ON armv7l...";\
		GOOS=linux GOARCH=arm GOARM=7 go build -o builds/dms_linux_arm32_v7l -ldflags=$(LDFLAGS) .;\
	elif command -v arm-linux-gnueabihf-gcc > /dev/null 2>&1; then\
		echo "Cross Compiling for armv7l...";\
		CGO_ENABLED=1 CC_FOR_TARGET=gcc-arm-linux-gnueabihf CC=arm-linux-gnueabihf-gcc GOOS=linux GOARCH=arm GOARM=7 go build -o builds/dms_linux_arm32_v7l -ldflags=$(LDFLAGS) .;\
	else\
		echo "arm-linux-gnueabihf - no compiler found";\
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

setcap_e2e: 
	sudo setcap cap_net_admin,cap_sys_admin+ep ./tests/e2e/dms

e2e:
	@echo "Running all e2e tests"
	@if ! docker image inspect nunet-glusterfs-client >/dev/null 2>&1; then \
		echo "Docker image nunet-glusterfs-client not found. Building..."; \
		make build-nunet-glusterfs-client; \
	fi
	go build -o ./tests/e2e/dms -ldflags=$(LDFLAGS)
	make setcap_e2e
	go test -v ./tests/e2e/... -tags=e2e -timeout=10m $(ARGS)

e2e-%:
	@echo "Running e2e test: TestE2E/$*"
	@if ! docker image inspect nunet-glusterfs-client >/dev/null 2>&1; then \
		echo "Docker image nunet-glusterfs-client not found. Building..."; \
		make build-nunet-glusterfs-client; \
	fi
	go build -o ./tests/e2e/dms -ldflags=$(LDFLAGS)
	make setcap_e2e
	go test -v ./tests/e2e/... -tags=e2e -timeout=10m -run "TestE2E/$*" $(ARGS)

run-acceptance: linux_amd64
	go test -test.v ./tests/acceptance/ -tags=acceptance -timeout=10m -godog.tags="~@wip"

build-nunet-glusterfs-client:
	docker build -t nunet-glusterfs-client storage/volume/glusterfs/client_image

build_storage_tests: 
	go test -tags storagetst -c ./tests/e2e/ -o ./tests/e2e/storagetestbinary

setcapstorage: 
	sudo setcap cap_net_admin,cap_sys_admin+ep ./tests/e2e/storagetestbinary
 
storage_test:
	@echo "Running storage test"
	make linux_amd64
	cp builds/dms_linux_amd64 tests/e2e/dms
	make build_storage_tests
	make setcapstorage
	./tests/e2e/storagetestbinary

generate:
	$(PROTOC) --proto_path=$(PROTO_DIR) --go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative $(PROTO_FILES) --go_opt=Mcommon.proto=proto/generated/common

# make generate-glusterfs-client-certs CN=did:key:here or clientA
generate-glusterfs-client-certs:
	@if [ -z "$(CN)" ]; then \
		echo "Error: CN variable is required. Usage: make generate-glusterfs-client-certs CN=<client_name>"; \
		exit 1; \
	fi
	@echo "Generating client certificates"
	mkdir glusterfs_certificates
	openssl genrsa -out glusterfs_certificates/glusterfs.key 2048
	openssl req -new -x509 -key glusterfs_certificates/glusterfs.key -subj "/CN=$(CN)" -out glusterfs_certificates/glusterfs.pem -days 365
	

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
