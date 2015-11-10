#!/bin/bash
set -ex

CIRCLEUTIL_TAG="v1.12"

export GOPATH_INTO="$HOME/installed_gotools"
export GOLANG_VERSION="1.5.1"
export GO15VENDOREXPERIMENT="1"
export GOROOT="$HOME/go_circle"
export GOPATH="$HOME/.go_circle"
export PATH="$GOROOT/bin:$GOPATH/bin:$GOPATH_INTO:$PATH"
export IMPORT_PATH="github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME"
export CIRCLE_ARTIFACTS="${CIRCLE_ARTIFACTS-/tmp}"

# Assumes that circleutil has been sourced
function docker_tag() {
  DOCKTAG=$(docker_release_tag "$CIRCLE_BRANCH")
  echo "quay.io/signalfx/phabricator-circleci:${DOCKTAG}$DOCKER_TAG_SUFFIX"
}

SRC_PATH="$GOPATH/src/github.com/$CIRCLE_PROJECT_USERNAME/$CIRCLE_PROJECT_REPONAME"

function do_cache() {
  [ ! -d "$HOME/circleutil" ] && git clone https://github.com/signalfx/circleutil.git "$HOME/circleutil"
  (
    cd "$HOME/circleutil"
    git fetch -a -v
    git fetch --tags
    git reset --hard $CIRCLEUTIL_TAG
  )
  . "$HOME/circleutil/scripts/common.sh"
  . "$HOME/circleutil/scripts/install_all_go_versions.sh"
  . "$HOME/circleutil/scripts/versioned_goget.sh" "github.com/cep21/gobuild:v1.0"
  copy_local_to_path "$SRC_PATH"
  (
    cd "$SRC_PATH"
    CGO_ENABLED=0 go build -v -installsuffix .
    docker build -t "$(docker_tag)"
  )
}

function do_test() {
  . "$HOME/circleutil/scripts/common.sh"
  copy_local_to_path "$SRC_PATH"
  (
    cd "$SRC_PATH"
    gobuild -verbose -verbosefile "$CIRCLE_ARTIFACTS/gobuildout.txt"
  )
}

function do_deploy() {
  . "$HOME/circleutil/scripts/common.sh"
  (
    cd "$SRC_PATH"
    if [ "$DOCKER_PUSH" == "1" ]; then
      docker push "$(docker_tag)"
    fi
  )
}

function do_all() {
  do_cache
  do_test
  do_deploy
}

case "$1" in
  cache)
    do_cache
    ;;
  test)
    do_test
    ;;
  deploy)
    do_deploy
    ;;
  all)
    do_all
    ;;
  *)
  echo "Usage: $0 {cache|test|deploy|all}"
    exit 1
    ;;
esac

