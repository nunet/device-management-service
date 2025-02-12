FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt update \
  && apt install -y \
    sudo \
    iptables \
    dnsutils \
    git \
    curl \
    wget \
    libc6 \
    libcap2-bin \
    make \
    build-essential \
    dpkg-dev \
    devscripts \
    lintian \
    libsystemd-dev \
    zip \
    gcc-arm-linux-gnueabihf \
    gcc-aarch64-linux-gnu \
  && apt autoremove -y && apt clean \
  && rm -rf /var/lib/apt/lists/*

# Install Go 1.22.7
RUN curl -LO https://go.dev/dl/go1.22.7.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf go1.22.7.linux-amd64.tar.gz \
  && rm go1.22.7.linux-amd64.tar.gz

# Set up Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

