FROM golang:1.22.7-bookworm

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update \
  && apt install git curl wget libc6 make build-essential dpkg-dev devscripts lintian libsystemd-dev zip -y \
  && apt install -y gcc-arm-linux-gnueabihf gcc-aarch64-linux-gnu \
  && apt autoremove -y && apt clean \
  && rm -rf /var/lib/apt/lists/*

