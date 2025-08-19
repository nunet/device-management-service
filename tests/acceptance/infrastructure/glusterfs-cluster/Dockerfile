FROM ubuntu:24.04

ENV TZ="Etc/UTC"
ENV PATH="$PATH:/root/.local/bin"
ARG DEBIAN_FRONTEND=noninteractive

RUN apt update && apt install -y --no-install-recommends \
  pipx \
  jq \
  incus \
  openssh-client \
  unzip \
  rsync \
  yq \
  make \
  golang \
  git \
  && rm -rf /var/lib/apt/lists/*
RUN pipx install --include-deps ansible
