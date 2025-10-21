# Copyright 2024, Nunet
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and limitations under the License.

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
  iptables \
  && rm -rf /var/lib/apt/lists/*
RUN pipx install --include-deps ansible
