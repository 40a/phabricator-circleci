#!/bin/bash
set -ex

pushd "$SRC_PATH"

if [ -z "$CIRCLE_ARTIFACTS" ]; then
	export CIRCLE_ARTIFACTS=/tmp
fi

mdl --warnings < README.md

env GOPATH="$GOPATH:$(godep path)" gobuild -verbose -verbosefile "$CIRCLE_ARTIFACTS/gobuildout.txt"
CGO_ENABLED=0 go build -v -installsuffix .
popd
