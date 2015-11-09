#!/bin/bash
set -ex

pushd "$SRC_PATH"

if [ -z "$CIRCLE_ARTIFACTS" ]; then
  export CIRCLE_ARTIFACTS=/tmp
fi

gobuild -verbose -verbosefile "$CIRCLE_ARTIFACTS/gobuildout.txt"
CGO_ENABLED=0 go build -v -installsuffix .
popd
