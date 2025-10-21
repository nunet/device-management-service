#!/bin/bash

DIR=$( dirname -- "$0" )

#set -x
go run $DIR/logs "$@" --query '.error'
