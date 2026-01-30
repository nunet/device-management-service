#!/usr/bin/env bash
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

GO_COVER_EXCLUSION_LIST=$(realpath "$SCRIPT_DIR/../go-coverage-exclude.txt")
GO_PKG_LIST=$(go list ./...)
if [[ -f "$GO_COVER_EXCLUSION_LIST" ]]; then
    echo exclusion list for unit tests found at $GO_COVER_EXCLUSION_LIST...
    while read -r folder_to_exclude; do
        echo excluding $folder_to_exclude...
        GO_PKG_LIST=$(echo "$GO_PKG_LIST" | grep -v "$folder_to_exclude")
    done <"$GO_COVER_EXCLUSION_LIST"
fi

set -euo pipefail

# This is akin to `defer` in go. Here it's used to preserve the exit code of
# the test so we can pass it down the pipeline
print-total-coverage() {
    echo coverage: "$(go tool cover -func=coverage.txt | grep total: | awk '{print $3}')" of statements
}
trap print-total-coverage EXIT

go install gotest.tools/gotestsum@latest
"${GOPATH:-$HOME/go}"/bin/gotestsum \
    --junitfile junit.xml \
    --format testname \
    -- -coverprofile=coverage.txt -covermode count -tags=unit $GO_PKG_LIST -timeout=30m
