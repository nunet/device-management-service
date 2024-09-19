FROM ubuntu:22.04

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install git curl wget libc6 make build-essential dpkg-dev devscripts lintian libsystemd-dev pandoc zip -y

# Install Cross Compilers for arm32 and aarch64 builds
RUN apt install -y gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu

# Golang install
RUN wget https://go.dev/dl/go1.21.7.linux-amd64.tar.gz
RUN tar -xf go1.21.7.linux-amd64.tar.gz
RUN mv go /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go
RUN ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
RUN rm go1.21.7.linux-amd64.tar.gz
