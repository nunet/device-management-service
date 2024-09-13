#!/bin/bash
set -euo pipefail

# Build process comprises of following steps:

# Supported architectures: amd64

# Requirements

# golang is required to build the nunet binary
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

    go version # redundant check of go version
    make linux_$arch

    # create bin only zip release
    zip -j $outputDir/nunet-dms_${version}_${arch}.zip builds/dms_linux_$arch

    cp builds/dms_linux_$arch $archDir/usr/bin/nunet
    ls -R $archDir/usr # to allow checking all files are where they're supposed to be

    # create man page
    pandoc -s -t man $archDir/usr/share/man/man1/nunet-cli-man.md -o $archDir/usr/share/man/man1/nunet.1
    gzip $archDir/usr/share/man/man1/nunet.1
    rm $archDir/usr/share/man/man1/nunet-cli-man.md

    DMS_INST_SIZE=$(du -sB1 $archDir | awk '{ print $1 }')
    sed -i "s/Installed-Size:.*/Installed-Size: $DMS_INST_SIZE/g" $archDir/DEBIAN/control

    find $archDir -name .gitkeep | xargs rm
    chmod -R 755 $archDir
    dpkg-deb --build --root-owner-group $archDir $outputDir
    rm -r $archDir

    # NUNETBOT_BUILD_ENDPOINT is only available for protected branches, therefore pipeline
    #   builds in gitlab ci from branches that aren't protected should also skip this step
    if [[ -v GITLAB_CI ]] && [[ -v NUNETBOT_BUILD_ENDPOINT ]] ; then
        # upload artifact from build.sh to GitLab Package Registry.
        curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${projectRoot}/dist/nunet-dms_${version}_${arch}.deb ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.deb
        curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${outputDir}/nunet-dms_${version}_${arch}.zip ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.zip
        # notify the bot about the build
        curl -X POST -H "Content-Type: application/json" -H "$HOOK_TOKEN_HEADER_NAME: $HOOK_TOKEN_HEADER_VALUE" -d "{\"project\" : \"DMS\", \"version\" : \"$version\", \"commit\" : \"$CI_COMMIT_SHA\", \"commit_msg\" : \"$(echo $CI_COMMIT_MESSAGE | sed "s/\"/'/g")\", \"package_url\" : \"${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.deb\"}" $NUNETBOT_BUILD_ENDPOINT
    fi 
done
