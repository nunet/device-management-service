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

# Default is vm; can override with: make run-acceptance INSTANCE_TYPE=container
INSTANCE_TYPE ?= vm

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
	
linux_amd64_docker:
	@echo "Building for Linux AMD64 using Docker..."
	make build-dms-builder
	docker run --rm \
		--env GOFLAGS=-buildvcs=false \
		--entrypoint="" \
		--workdir /app \
		-v $(PWD):/app \
		dms-builder \
		bash -c '\
			git config --global --add safe.directory /app && \
			go mod tidy && \
			CGO_ENABLED=1 CC_FOR_TARGET=gcc-aarch64-linux-gnu CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o builds/dms_linux_amd64 -ldflags=$(LDFLAGS) .;\
		'

BUILD_ARCHS := "amd64 arm64 arm32_v6l arm32_v7l"
dist-linux:
	make build-dms-builder
	docker run --rm \
		--env GOFLAGS=-buildvcs=false \
		--env BUILD_ARCHS=$(BUILD_ARCHS) \
		--entrypoint="" \
		--workdir /app \
		-v $(PWD):/app \
		dms-builder \
		bash -c 'git config --global --add safe.directory /app && go mod tidy && bash maint-scripts/build.sh'

dist-%:
	make dist-linux BUILD_ARCHS="$*"

linux_amd64_debug:
	@echo "Building for Linux AMD64 with debug..."
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o builds/dms_linux_amd64 -ldflags=$(LDFLAGS) -gcflags="all=-N -l" .

linux_amd64_e2e:
	make linux_amd64_debug
	mv builds/dms_linux_amd64 tests/e2e/dms

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

format:
	gofumpt -w .

clean:
	@echo "Cleaning up..."
	sudo rm -rf builds/ dist/

setcap_e2e: 
	sudo setcap cap_net_admin,cap_sys_admin+ep ./tests/e2e/dms

build-dms-builder:
	@echo "Building dms-builder docker image"
	docker build -f $(PWD)/maint-scripts/Dockerfile.builder -t dms-builder $(PWD)/maint-scripts

unit-docker:
	make build-dms-builder
	docker build -f $(PWD)/maint-scripts/Dockerfile.unit-tests -t dms-unit-tests $(PWD)/maint-scripts
	git lfs install && git lfs fetch && git lfs pull
	make testdata
	docker run -it --rm \
		--name dms-unit-tests \
		--env GOFLAGS=-buildvcs=false \
		--entrypoint="" \
		--workdir /app \
		-v $(PWD):/app \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v /tmp/nunet:/tmp/nunet \
		-v /root/.cache:/tmp/dms-unit-tests/cache \
		dms-unit-tests \
		bash -c 'git config --global --add safe.directory /app && bash /app/maint-scripts/unit-tests.sh'

unit:
	git lfs install && git lfs fetch && git lfs pull
	make testdata
	export GOFLAGS=-buildvcs=false
	bash $(PWD)/maint-scripts/unit-tests.sh

e2e:
	@echo "Running all e2e tests"
	@if ! docker image inspect nunet-glusterfs-client >/dev/null 2>&1; then \
		echo "Docker image nunet-glusterfs-client not found. Building..."; \
		make build-nunet-glusterfs-client; \
	fi
	go build -o ./tests/e2e/dms -ldflags=$(LDFLAGS)
	make setcap_e2e
	go test -failfast -v ./tests/e2e/... -tags=e2e -timeout=15m $(ARGS)

e2e-%:
	@echo "Running e2e test: TestE2E/$*"
	@if ! docker image inspect nunet-glusterfs-client >/dev/null 2>&1; then \
		echo "Docker image nunet-glusterfs-client not found. Building..."; \
		make build-nunet-glusterfs-client; \
	fi
	go build -o ./tests/e2e/dms -ldflags=$(LDFLAGS)
	make setcap_e2e
	go test -failfast -v ./tests/e2e/... -tags=e2e -timeout=15m -run "TestE2E/$*" $(ARGS)

run-acceptance:
	@echo "Running acceptance tests"
	INSTANCE_TYPE=$(INSTANCE_TYPE) go test -test.v ./tests/acceptance/ -tags=acceptance -timeout=25m -godog.tags="~@wip"

run-acceptance-%:
	@echo "Running acceptance tests: $*"
	INSTANCE_TYPE=$(INSTANCE_TYPE) go test -test.v ./tests/acceptance/ -tags=acceptance -timeout=25m -godog.tags="~@wip" -test.run "^$*/"

run-acceptance-container:
	make run-acceptance INSTANCE_TYPE=container

run-acceptance-vm:
	make run-acceptance INSTANCE_TYPE=vm

run-acceptance-docker:
	make build-acc-test-runner-image
	docker run -it --rm \
		--network host \
		-v $(PWD):/app \
		-v /var/lib/incus/unix.socket:/var/lib/incus/unix.socket \
		--workdir /app \
		nunet-acc-test-runner \
		bash -c 'make run-acceptance'

build-and-run-acceptance:
	@if [ $(UNAME) = Linux ]; then\
		make linux_amd64;\
	elif [ $(UNAME) = Darwin ]; then\
		make linux_amd64_docker;\
	fi
	make run-acceptance

build-nunet-glusterfs-client:
	docker build -t nunet-glusterfs-client storage/volume/glusterfs/client_image

build-acc-test-runner-image:
	docker build -t nunet-acc-test-runner tests/acceptance/infrastructure/glusterfs-cluster

provision-acc-tests-infra:
	make clean
	make dist-amd64
	make reprovision-acc-tests-infra

reprovision-acc-tests-infra:
	make build-acc-test-runner-image
	@$(eval ACC_TEST_DMS_DEB_FILE := $(shell realpath dist/nunet-dms*amd64.deb | head -n1))
	@echo Provisioning the glusterfs cluster for the acceptance tests using $(ACC_TEST_DMS_DEB_FILE)
	docker run -it --rm \
		--network host \
		-v $(PWD):/app \
		-v /var/lib/incus/unix.socket:/var/lib/incus/unix.socket \
		-v $(ACC_TEST_DMS_DEB_FILE):$(ACC_TEST_DMS_DEB_FILE) \
		--env ACC_TEST_DMS_DEB_FILE=$(ACC_TEST_DMS_DEB_FILE) \
		--workdir /app/tests/acceptance/infrastructure/glusterfs-cluster  \
		nunet-acc-test-runner \
		bash -c 'bash clear-and-launch.sh && ssh-agent bash run.sh'

deprovision-acc-tests-infra:
	docker run -it --rm \
		--network host \
		-v $(PWD):/app \
		-v /var/lib/incus/unix.socket:/var/lib/incus/unix.socket \
		--workdir /app/tests/acceptance/infrastructure/glusterfs-cluster  \
		nunet-acc-test-runner \
		bash -c 'source lib.sh && clear_glusterfs_vms'

test-acc-tests-infra:
	docker run -it --rm \
		--network host \
		-v $(PWD):/app \
		-v /var/lib/incus/unix.socket:/var/lib/incus/unix.socket \
		--workdir /app/tests/acceptance/infrastructure/glusterfs-cluster  \
		nunet-acc-test-runner \
		bash -c 'ssh-agent bash run.sh test glusterfs && ssh-agent bash run.sh test dms'

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


# use for downloading data or setups required before running tests
testdata: $(testdata_objects)
	@echo "Preparing test data..."
	@echo "Nothing to do at the moment."