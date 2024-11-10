# Introduction

This directory contains utility scripts for building / development assistance and runtime; It is included into final build;

There is a systmed unit file in the debian package but it's not enabled or started by the post installation script because the DMS daemon requires a passphrase on `run`. It is possible to assign the passphrase to env var $DMS_PASSPHRASE and add it to the unit file but that's not recommended.

_Note: lets see if this functionality can be split to relevant packages and leave only the functionality that cannot be moved elsewhere;_

# Summary

There are 3 files/subdirectory in this directory. Here are what they are for:

1. `nunet-dms/`: This is a template directory to build deb file. This direcotry is used by `build.sh` to write the binary file in the `nunet-dms_$version_$arch/usr/bin`, update architecture and version in the control file. And then build a .deb file out of the direcotry.

2. `build.sh`: This script is intended to be used by CI/CD server. This script creates .deb package for `amd64` and `arm64`.

3. `clean.sh`: This script is intended to be used by developers. You should be using the `apt remove nunet-dms` otherwise. Use this clean script only if installation is left broken.


---

## How to build locally

### Requirements/dependencies:

- `golang` is required to build the _nunet_ binary
- `dpkg-deb` is required to build the debian package

### Invoking a build

Build is supposed to be invoked from the root of the project. Please comment out the publish command from the build script, it is intended to be called from a GitLab CI environment and will fail locally.

A build can be invoked by:

    bash maint-script/build.sh
