#!/bin/bash

usage() {
    echo "Usage: $0 [-t test_name] [-n node_number]"
    echo "  -t: Test name (default: deployment_tests)"
    echo "  -n: Node number (default: *)"
    exit 1
}

TEST="deployment_tests"
NODE="*"

while getopts "t:n:h" opt; do
    case $opt in
        t) TEST="$OPTARG" ;;
        n) NODE="$OPTARG" ;;
        h) usage ;;
        ?) usage ;;
    esac
done

NODE=dms$NODE

LOG_PATH=tests/e2e/testdata/${TEST}/${NODE}/logs.jsonl

for log_file in $LOG_PATH; do
    echo "##### ##### #####"
    echo "DMS logs: $log_file"
    echo "##### ##### #####"
    if [ -f "$log_file" ]; then
        fblog "$log_file" -a did -a error
    fi
    echo ""
done