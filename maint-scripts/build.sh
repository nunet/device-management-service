#!/bin/bash
set -euo pipefail

# Build process comprises of following steps:

# Supported architectures: amd64

# INSTALLATION PROCESS

# 1. Install Docker.
# 2. Create a nunet user. This user will be used to run the device-management-service. This user will have access to write to /etc/nunet.
# 3. Get nunet-dms.service. Copy it to appropriate location. Make sure it is run at the end of the installation process.

# UNINSTALLATION PROCESS
# 1. Stop services and remove service files.

# Both INSTALLATION PROCESS and UNINSTALLATION PROCESS are done by the package created in the build process.

# Requirements

# golang is required to build the nunet binary
# gcc is required to build the nunet-tap-config binary
# dpkg-deb is required to create the .deb package
# pandoc is required to convert the markdown into a man page

projectRoot=$(pwd)
outputDir="$projectRoot/dist"

# TODO: Update version number from git describe after release
fullVersion="v0.5.0-boot"
version=$(echo $fullVersion | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')

mkdir -p $outputDir

for arch in amd64 # arm64
do
    # echo .deb file will be written to: $outputDir
    archDir=$projectRoot/maint-scripts/nunet-dms_$fullVersion\_$arch
    cp -r $projectRoot/maint-scripts/nunet-dms $archDir
    sed -i "s/Version:.*/Version: $version/g" $archDir/DEBIAN/control
    sed -i "s/Architecture:.*/Architecture: $arch/g" $archDir/DEBIAN/control

    DMS_INST_SIZE=$(du -sB1 $archDir | awk '{ print $1 }')
    sed -i "s/Installed-Size:.*/Installed-Size: $DMS_INST_SIZE/g" $archDir/DEBIAN/control

    go version # redundant check of go version
    make linux_$arch
    cp builds/dms_linux_$arch $archDir/usr/bin/nunet
    ls -R $archDir/usr # to allow checking all files are where they're supposed to be

    gcc $projectRoot/maint-scripts/config_network.c -o $archDir/usr/bin/nunet-tap-config

    # start including firecracker
    curl -L https://github.com/firecracker-microvm/firecracker/releases/download/v1.1.1/firecracker-v1.1.1-x86_64.tgz -o $archDir/firecracker-v1.1.1-x86_64.tgz
    tar -xf $archDir/firecracker-v1.1.1-x86_64.tgz -C $archDir/
    mv -v $archDir/release-v1.1.1-x86_64/firecracker-v1.1.1-x86_64 $archDir/usr/bin/firecracker
    rm -rf $archDir/release-v1.1.1-x86_64/
    rm -rf $archDir/firecracker-v1.1.1-x86_64.tgz
    # including firecracker ends

    # download websocat 
    wget -qO $archDir/usr/bin/websocat https://github.com/vi/websocat/releases/latest/download/websocat.x86_64-unknown-linux-musl
    chmod a+x $archDir/usr/bin/websocat

    # create man page
    pandoc -s -t man $archDir/usr/share/man/man1/NUNET-CLI-MANUAL.md -o $archDir/usr/share/man/man1/nunet.1
    gzip $archDir/usr/share/man/man1/nunet.1
    rm $archDir/usr/share/man/man1/NUNET-CLI-MANUAL.md

    find $archDir -name .gitkeep | xargs rm
    chmod -R 755 $archDir
    dpkg-deb --build --root-owner-group $archDir $outputDir
    rm -r $archDir

    # The remaining part of this script used to upload artifact from build.sh to GitLab Package Registry.
    #
    # NUNETBOT_BUILD_ENDPOINT is only available for protected branches, therefore pipeline
    #   builds in gitlab ci from branches that aren't protected should also skip this step
    if [[ -v GITLAB_CI ]] && [[ -v NUNETBOT_BUILD_ENDPOINT ]] ; then
        curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${projectRoot}/dist/nunet-dms_${version}_${arch}.deb ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.deb
        curl -X POST -H "Content-Type: application/json" -H "$HOOK_TOKEN_HEADER_NAME: $HOOK_TOKEN_HEADER_VALUE" -d "{\"project\" : \"DMS\", \"version\" : \"$version\", \"commit\" : \"$CI_COMMIT_SHA\", \"commit_msg\" : \"$(echo $CI_COMMIT_MESSAGE | sed "s/\"/'/g")\", \"package_url\" : \"${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.deb\"}" $NUNETBOT_BUILD_ENDPOINT
    fi 
done
