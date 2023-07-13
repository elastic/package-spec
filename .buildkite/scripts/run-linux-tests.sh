#!/bin/bash

set -euo pipefail

source .buildkite/scripts/install_deps.sh

install_go_dependencies

make test-ci
