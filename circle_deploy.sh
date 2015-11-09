#!/bin/bash
set -ex

pushd "$SRC_PATH"

export DOCKER_TAG=$(echo "$CIRCLE_BRANCH" | sed -e 's#.*/##')
if [ "$DOCKER_TAG" = "latest" ]; then
  DOCKER_TAG="latest-branch"
fi
if [ "$DOCKER_TAG" = "release" ]; then
  export DOCKER_TAG="latest"
fi
if [ "$CIRCLE_PROJECT_REPONAME" = "phabricator-circleci" ]; then
  export DOCKER_PUSH=$DOCKER_PUSH_ENABLED
fi
export COMMIT_SHA=$CIRCLE_SHA1


if [ -z "$CIRCLE_ARTIFACTS" ]; then
  export CIRCLE_ARTIFACTS=/tmp
fi

docker rmi "quay.io/signalfx/phabricator-circleci:${DOCKER_TAG}" || true
docker build -t "quay.io/signalfx/phabricator-circleci:${DOCKER_TAG}" .
if [ "$DOCKER_PUSH" == "1" ]; then
  docker push "quay.io/signalfx/phabricator-circleci:${DOCKER_TAG}"
fi

popd
