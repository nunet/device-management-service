PROTO_DIR := proto/v1/common
GO_OUT_DIR := proto/generated/v1/common

PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

PROTOC := protoc

generate:
	$(PROTOC) --proto_path=$(PROTO_DIR) --go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative $(PROTO_FILES) --go_opt=Mcommon.proto=proto/generated/common

.PHONY: all clean linux_amd64 darwin_arm64

all: linux_amd64 darwin_arm64

linux_amd64:
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -o builds/dms_linux_amd64 .

darwin_arm64:
	@echo "Building for Darwin ARM64..."
	GOOS=darwin GOARCH=arm64 go build -o builds/dms_darwin_arm64 .

darwin_amd64:
	@echo "Building for Darwin AMD64..."
	GOOS=darwin GOARCH=amd64 go build -o builds/dms_darwin_amd64 .

clean:
	@echo "Cleaning up..."
	rm -rf builds/

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
