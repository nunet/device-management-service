#!/bin/bash
set -xeuo pipefail

# Build process comprises of following steps:

# Supported architectures: amd64

# Requirements

# golang is required to build the nunet binary
# dpkg-deb is required to create the .deb package

projectRoot=$(pwd)
outputDir="$projectRoot/dist"

noCommits=$(git rev-list $(git describe --tags --always --abbrev=0)..HEAD --count)
fullVersion=$(git describe --tags --always --abbrev=0 --dirty)-$noCommits-$(git rev-parse --short HEAD)
version=$(echo $fullVersion | cut -c 2-)

mkdir -p $outputDir

BUILD_ARCHS="${BUILD_ARCHS:-amd64 arm64 arm32_v6l arm32_v7l}"
for arch in $BUILD_ARCHS; do

    debarch=$arch
    if [[ $arch == "arm32_v6l" ]]; then
        debarch="armel"
    elif [[ $arch == "arm32_v7l" ]]; then
        debarch="armhf"
    fi
    archDir=$projectRoot/maint-scripts/nunet-dms_$fullVersion\_linux_$debarch
    cp -r $projectRoot/maint-scripts/nunet-dms $archDir
    sed -i "s/Version:.*/Version: $version/g" $archDir/DEBIAN/control
    sed -i "s/Architecture:.*/Architecture: $debarch/g" $archDir/DEBIAN/control

    go version # redundant check of go version
    make linux_$arch

    # check if bin is created successfully
    if [ ! -f builds/dms_linux_$arch ]; then
        echo "Error: builds/dms_linux_$arch not found"
        rm -r $archDir
        continue
    fi

    # create bin only zip release
    zip -j $outputDir/nunet-dms_${fullVersion}_${debarch}.zip builds/dms_linux_$arch

    cp builds/dms_linux_$arch $archDir/usr/bin/nunet
    ls -R $archDir/usr # to allow checking all files are where they're supposed to be

    # create man page
    gzip $archDir/usr/share/man/man1/nunet.1

    DMS_INST_SIZE=$(du -sB1024 $archDir | awk '{ print $1 }')
    sed -i "s/Installed-Size:.*/Installed-Size: $DMS_INST_SIZE/g" $archDir/DEBIAN/control

    find $archDir -name .gitkeep | xargs rm
    chmod -R 755 $archDir

    dpkg-deb --build --root-owner-group $archDir $outputDir
    rm -r $archDir

    if [[ -n ${GITLAB_CI+x} ]]; then
        regex='^(release|release\/.+|release-.+|patch\/.+|patch-.+|main)$'

        if [[ $CI_COMMIT_REF_NAME =~ $regex ]]; then
            echo "Deploying to package registry..."
            curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${projectRoot}/dist/nunet-dms_${version}_${debarch}.deb ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${fullVersion}/nunet-dms_${fullVersion}_linux_${debarch}.deb
            curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${outputDir}/nunet-dms_${fullVersion}_${debarch}.zip ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${fullVersion}/nunet-dms_${fullVersion}_linux_${debarch}.zip
        else
            echo skipping deployment to package registry...
        fi
    fi

    if [[ -n ${NUNETBOT_BUILD_ENDPOINT+x} ]]; then
        # notify the bot about the build
        curl -X POST -H "Content-Type: application/json" -H "$HOOK_TOKEN_HEADER_NAME: $HOOK_TOKEN_HEADER_VALUE" -d "{\"project\" : \"DMS\", \"version\" : \"$fullVersion\", \"commit\" : \"$CI_COMMIT_SHA\", \"commit_msg\" : \"$(echo $CI_COMMIT_MESSAGE | sed "s/\"/'/g")\", \"package_url\" : \"${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${fullVersion}/nunet-dms_${fullVersion}_${debarch}.deb\"}" $NUNETBOT_BUILD_ENDPOINT
    fi
done
