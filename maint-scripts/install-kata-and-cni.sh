#!/usr/bin/env bash
# Install Kata Containers (GitHub release) and CNI plugins.
# This script does not modify containerd config; it prints instructions instead.
# Usage: sudo ./maint-scripts/install-kata.sh [kata-version]
# Example: sudo ./maint-scripts/install-kata.sh 3.31.0

set -euo pipefail

KATA_REPO="kata-containers/kata-containers"
CNI_REPO="containernetworking/plugins"
KATA_INSTALL_DIR="/opt/kata"
CNI_BIN_DIR="/opt/cni/bin"
CONTAINERD_CONFIG="/etc/containerd/config.toml"
LINK_DIR="/usr/local/bin"
TMPDIR="${TMPDIR:-/tmp}/nunet-kata-install-$$"

step() { echo "==> $*"; }

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Run as root (or with sudo)." >&2
	exit 1
fi

case "$(uname -m)" in
x86_64) ARCH=amd64 ;;
aarch64) ARCH=arm64 ;;
ppc64le) ARCH=ppc64le ;;
s390x) ARCH=s390x ;;
*)
	echo "Unsupported architecture: $(uname -m)" >&2
	exit 1
	;;
esac

if [[ $# -ge 1 ]]; then
	KATA_VERSION="${1#v}"
else
	step "Resolving latest Kata release"
	KATA_TAG="$(curl -fsSL "https://api.github.com/repos/${KATA_REPO}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
	KATA_VERSION="${KATA_TAG#v}"
fi

step "Platform: $(uname -m) (${ARCH}), Kata version: ${KATA_VERSION}"

mkdir -p "$TMPDIR"

if ! command -v zstd >/dev/null 2>&1; then
	echo "zstd is not installed." >&2
	echo "Install zstd before proceeding (required by some Kata release archives)." >&2
	exit 1
fi

step "Downloading Kata static bundle"
KATA_URL="$(curl -fsSL "https://api.github.com/repos/${KATA_REPO}/releases/tags/${KATA_VERSION}" \
	| grep -oE "https://[^\"]+kata-static-${KATA_VERSION}-${ARCH}\.tar\.(xz|zst)" | head -1)"
if [[ -z "$KATA_URL" ]]; then
	# Some tags use a leading v in the release tag while assets omit it.
	KATA_URL="$(curl -fsSL "https://api.github.com/repos/${KATA_REPO}/releases/tags/v${KATA_VERSION}" \
		| grep -oE "https://[^\"]+kata-static-${KATA_VERSION}-${ARCH}\.tar\.(xz|zst)" | head -1)"
fi
[[ -n "$KATA_URL" ]] || { echo "Kata tarball not found for ${KATA_VERSION} (${ARCH})" >&2; exit 1; }

KATA_TAR="$TMPDIR/$(basename "$KATA_URL")"
curl -fSL --progress-bar -o "$KATA_TAR" "$KATA_URL"

step "Extracting Kata to /"
tar -xf "$KATA_TAR" -C /


step "Linking Kata binaries into ${LINK_DIR}"
mkdir -p "$LINK_DIR"
for bin in containerd-shim-kata-v2 kata-runtime kata-collect-data.sh; do
	ln -sf "${KATA_INSTALL_DIR}/bin/${bin}" "${LINK_DIR}/${bin}"
done

step "Downloading CNI plugins"
CNI_TAG="$(curl -fsSL "https://api.github.com/repos/${CNI_REPO}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
CNI_URL="$(curl -fsSL "https://api.github.com/repos/${CNI_REPO}/releases/tags/${CNI_TAG}" \
	| grep -oE "https://[^\"]+cni-plugins-linux-${ARCH}-${CNI_TAG}\.tgz" | head -1)"
[[ -n "$CNI_URL" ]] || { echo "CNI plugins tarball not found for ${CNI_TAG} (${ARCH})" >&2; exit 1; }

mkdir -p "$CNI_BIN_DIR"
curl -fSL --progress-bar "$CNI_URL" | tar -xz -C "$CNI_BIN_DIR"

if command -v containerd >/dev/null 2>&1; then
	step "containerd is installed"
else
	step "containerd is not installed"
fi

KATA_CONTAINERD_SNIPPET="$(cat <<EOF
[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "kata"
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
          runtime_type = "io.containerd.kata.v2"
EOF
)"

if [[ -f "$CONTAINERD_CONFIG" ]] && grep -q 'io.containerd.kata.v2' "$CONTAINERD_CONFIG"; then
	step "containerd config already contains io.containerd.kata.v2 runtime"
	echo "No config changes needed for Kata runtime in ${CONTAINERD_CONFIG}."
else
	echo
	echo "Containerd setup instructions:"
	if ! command -v containerd >/dev/null 2>&1; then
		echo "1) Install containerd using your OS package manager or from https://github.com/containerd/containerd/releases"
		echo "2) Add the Kata config below to ${CONTAINERD_CONFIG}. Also visit the instructions at https://github.com/kata-containers/kata-containers/blob/main/docs/install/container-manager/containerd/containerd-install.md"
		echo "3) Restart containerd after editing: sudo systemctl restart containerd"
	else
		echo "1) Edit ${CONTAINERD_CONFIG} and append the Kata runtime block below."
		echo "2) Restart containerd after editing: sudo systemctl restart containerd"
	fi
	echo
	echo "----- begin containerd config snippet -----"
	echo "${KATA_CONTAINERD_SNIPPET}"
	echo "----- end containerd config snippet -----"
	echo
fi

step "Done"
echo "Kata:    $(command -v containerd-shim-kata-v2)"
echo "CNI:     ${CNI_BIN_DIR} ($(ls "$CNI_BIN_DIR" | wc -l) plugins)"
echo "Runtime: io.containerd.kata.v2 (apply config snippet manually if needed)"
