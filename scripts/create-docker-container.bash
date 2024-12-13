#!/bin/bash

set -eu
set -o pipefail

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/git-helpers.bash"
REPO_NAME=$(git_get_remote_name)
REPO_PATH="${THIS_FILE_DIR}/../"
unset THIS_FILE_DIR

IMAGE="cloudfoundry/tas-runtime-build"
CONTAINER_NAME="$REPO_NAME-docker-container"

if [[ -z "${*}" ]]; then
  ARGS="-it"
else
  ARGS="${*}"
fi

if [[ -f "${HOME}/.bash_functions" ]]; then
  . "${HOME}/.bash_functions"
  export DOCKER_REGISTRY_USERNAME="$(gimme-secret-value-only dockerhub-tasruntime | yq -r .user)"
  export DOCKER_REGISTRY_PASSWORD="$(gimme-secret-value-only dockerhub-tasruntime | yq -r .password)"
  export PRIVATE_DOCKER_IMAGE_URL="docker://cloudfoundry/garden-private-image-test:groot"
fi

if [[ "${DOCKER_REGISTRY_USERNAME:-undefined}" == "undefined" || "${DOCKER_REGISTRY_PASSWORD:-undefined}" == "undefined" || "${PRIVATE_DOCKER_IMAGE_URL}:-undefined" == "undefined" ]]; then
  cat << EOF
  Please provide a private image for running tests in this repo. Build and push the docker image from "fetcher/layerfetcher/source/assets/groot-private-docker-image/Dockerfile"

  docker build -t my-user/my-image:my-tag fetcher/layerfetcher/source/assets/groot-private-docker-image
  docker login
  docker push my-user/my-image:my-tag

  Run this script with DOCKER_REGISTRY_USERNAME, DOCKER_REGISTRY_PASSWORD, and PRIVATE_DOCKER_IMAGE_URL env variables
EOF
exit 1
fi

docker pull "${IMAGE}"
docker rm -f $CONTAINER_NAME
docker run -it \
  --env "REPO_NAME=$REPO_NAME" \
  --env "REPO_PATH=/repo" \
  --env "DOCKER_REGISTRY_USERNAME=$DOCKER_REGISTRY_USERNAME" \
  --env "DOCKER_REGISTRY_PASSWORD=$DOCKER_REGISTRY_PASSWORD" \
  --env "PRIVATE_DOCKER_IMAGE_URL=$PRIVATE_DOCKER_IMAGE_URL" \
  --rm \
  --name "$CONTAINER_NAME" \
  -v "${REPO_PATH}:/repo" \
  -v "${CI}:/ci" \
  ${ARGS} \
  "${IMAGE}" \
  /bin/bash

