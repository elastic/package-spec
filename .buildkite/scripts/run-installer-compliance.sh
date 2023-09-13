#!/bin/bash

set -euo pipefail

WORKSPACE="$(pwd)"

source .buildkite/scripts/install_deps.sh
add_bin_path

echo "--- Install Go :go:"
with_go

echo "--- Install docker-compose"
with_docker_compose

function usage() {
    echo "usage: $0 STACK_VERSION SPEC_VERSION"
    exit 1
}

STACK_VERSION=$1
[[ -n "$STACK_VERSION" ]] || usage

SPEC_VERSION=$2
[[ -n "$SPEC_VERSION" ]] || usage

function start_stack() {
    local stack_version=$1
    local elastic_package="go run github.com/elastic/elastic-package"

    cd compliance
    $elastic_package stack up -d --version $stack_version
    eval $($elastic_package stack shellinit)
    cd -
}

echo "--- Start local Elastic Stack $STACK_VERSION with elastic-package"
start_stack $STACK_VERSION

echo "--- Check compliance with Package Spec $SPEC_VERSION"
TEST_SPEC_VERSION=$SPEC_VERSION TEST_SPEC_JUNIT=report-$STACK_VERSION-$SPEC_VERSION.xml make -C compliance test
