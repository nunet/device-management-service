FROM golang:1.22.7-bookworm

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update \
  && apt install -y \
    sudo \
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

