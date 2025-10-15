FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN apt update \
  && apt install -y \
    build-essential \
    curl \
    devscripts \
    dnsutils \
    dpkg-dev \
    gcc-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    gcc-x86-64-linux-gnu \
    git \
    git-lfs \
    iptables \
    libc6 \
    libcap2-bin \
    libsystemd-dev \
    lintian \
    make \
    sudo \
    wget \
    zip \
  && apt autoremove -y && apt clean \
  && rm -rf /var/lib/apt/lists/*

# Install Go 1.25.2
RUN curl -LO https://go.dev/dl/go1.25.2.linux-amd64.tar.gz \
  && tar -C /usr/local -xzf go1.25.2.linux-amd64.tar.gz \
  && rm go1.25.2.linux-amd64.tar.gz

# Set up Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

