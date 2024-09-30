#!/bin/bash
set -xeuo pipefail

# Build process comprises of following steps:

# Supported architectures: amd64

# Requirements

# golang is required to build the nunet binary
# pandoc is required to convert the markdown into a man page

projectRoot=$(pwd)
outputDir="$projectRoot/dist"

noCommits=$(git rev-list $(git describe --tags --abbrev=0)..HEAD --count)
fullVersion=$(git describe --tags --abbrev=0 --dirty)-$noCommits
version="$(echo $fullVersion | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')-${noCommits}"

mkdir -p $outputDir

for arch in arm64 # amd64
do
    archDir=$projectRoot/maint-scripts/nunet-dms_$fullVersion\_$arch
    cp -r $projectRoot/maint-scripts/nunet-dms $archDir

    go version # redundant check of go version
    make darwin_$arch

    # create bin only zip release
    zip -j $outputDir/nunet-dms_${version}_${arch}.zip builds/dms_darwin_$arch

    cp builds/dms_darwin_$arch $archDir/usr/bin/nunet
    ls -R $archDir/usr # to allow checking all files are where they're supposed to be

    # create man page
    pandoc -s -t man $archDir/usr/share/man/man1/nunet-cli-man.md -o $archDir/usr/share/man/man1/nunet.1
    gzip $archDir/usr/share/man/man1/nunet.1
    rm $archDir/usr/share/man/man1/nunet-cli-man.md

    find $archDir -name .gitkeep | xargs rm
    chmod -R 755 $archDir
    rm -r $archDir

    if [[ ! -z ${GITLAB_CI+x} ]]  ; then
        # upload artifact from build.sh to GitLab Package Registry.
        curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file ${outputDir}/nunet-dms_${version}_${arch}.zip ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${fullVersion}/nunet-dms_${fullVersion}_darwin_${arch}.zip
    fi

    if [[ ! -z ${NUNETBOT_BUILD_ENDPOINT+x} ]] ; then
        # notify the bot about the build
        curl -X POST -H "Content-Type: application/json" -H "$HOOK_TOKEN_HEADER_NAME: $HOOK_TOKEN_HEADER_VALUE" -d "{\"project\" : \"DMS\", \"version\" : \"$version\", \"commit\" : \"$CI_COMMIT_SHA\", \"commit_msg\" : \"$(echo $CI_COMMIT_MESSAGE | sed "s/\"/'/g")\", \"package_url\" : \"${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/nunet-dms/${version}/nunet-dms_${version}_${arch}.zip\"}" $NUNETBOT_BUILD_ENDPOINT
    fi
done

