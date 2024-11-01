// Original Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// Modified Copyright 2024, NuNet;
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//       http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// NuNet modifications:
//
// Inserting custom scripts to be executed from within Firecracker when booting the VM
// with `/sbin/overlay-init`

package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	overlayInit = `
#!/bin/sh

# Parameters:
# 1. rw_root -- path where the read/write root is mounted
# 2. work_dir -- path to the overlay workdir (must be on same filesystem as rw_root)
# Overlay will be set up on /mnt, original root on /mnt/rom
pivot() {
    local rw_root work_dir
    rw_root="$1"
    work_dir="$2"
    /bin/mount \
	-o noatime,lowerdir=/,upperdir=${rw_root},workdir=${work_dir} \
	-t overlay "overlayfs:${rw_root}" /mnt
    pivot_root /mnt /mnt/rom
}

# Overlay is configured under /overlay
# Global variable $overlay_root is expected to be set to either:
# "ram", which configures a tmpfs as the rw overlay layer (this is
# the default, if the variable is unset)
# - or -
# A block device name, relative to /dev, in which case it is assumed
# to contain an ext4 filesystem suitable for use as a rw overlay
# layer. e.g. "vdb"
do_overlay() {
    local overlay_dir="/overlay"
    if [ "$overlay_root" = ram ] ||
           [ -z "$overlay_root" ]; then
        /bin/mount -t tmpfs -o noatime,mode=0755 tmpfs /overlay
    else
        /bin/mount -t ext4 "/dev/$overlay_root" /overlay
    fi
    mkdir -p /overlay/root /overlay/work
    pivot /overlay/root /overlay/work
}

# If we're given an overlay, ensure that it really exists. Panic if not.
if [ -n "$overlay_root" ] &&
       [ "$overlay_root" != ram ] &&
       [ ! -b "/dev/$overlay_root" ]; then
    echo -n "FATAL: "
    echo "Overlay root given as $overlay_root but /dev/$overlay_root does not exist"
    exit 1
fi

do_overlay

# firecracker-containerd itself doesn't need /volumes but volume package
# uses that to share files between in-VM snapshotters.
mkdir /volumes
    `
)

func prepareCustomInit(initScripts map[string][]byte, customInitPath, initScriptsDir string) error {
	customInitContent := overlayInit

	customInitContent += "\n# (NuNet) Execute init scripts\n"
	customInitContent += fmt.Sprintf("for script in %s/*; do\n", initScriptsDir)
	customInitContent += "    if [ -x \"$script\" ]; then\n"
	customInitContent += "        \"$script\"\n"
	customInitContent += "    fi\n"
	customInitContent += "done\n\n"

	customInitContent += "exec /usr/sbin/init $@\n"

	err := os.WriteFile(customInitPath, []byte(customInitContent), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write custom init: %w", err)
	}

	// TODO: is 0755 necessary here?
	err = os.MkdirAll(initScriptsDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create init scripts directory: %w", err)
	}

	for name, content := range initScripts {
		err = os.WriteFile(filepath.Join(initScriptsDir, name), content, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write init script %s: %w", name, err)
		}
	}

	return nil
}
