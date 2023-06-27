#!/bin/bash

set -euo pipefail

source .buildkite/scripts/tooling.sh

add_bin_path(){
    mkdir -p ${WORKSPACE}/bin
    export PATH="${WORKSPACE}/bin:${PATH}"
}

create_workspace() {
    local path=${1}
    if [[ ! -d "${WORKSPACE}/${path}" ]]; then
    mkdir -p ${WORKSPACE}/${path}
    fi
}

with_go() {
    create_workspace "bin"
    retry 5 curl -sL -o ${WORKSPACE}/bin/gvm "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-linux-amd64"
    chmod +x ${WORKSPACE}/bin/gvm
    eval "$(gvm $(cat .go-version))"
    go version
    which go
    export PATH="$(go env GOPATH)/bin:${PATH}"
}

with_github_cli() {
    create_workspace "bin"
    create_workspace "tmp"

    local gh_filename="gh_${GH_CLI_VERSION}_linux_amd64"
    local gh_tar_file="${gh_filename}.tar.gz"
    local gh_tar_full_path="${WORKSPACE}/tmp/${gh_tar_file}"

    retry 5 curl -sL -o ${gh_tar_full_path} "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/${gh_tar_file}"

    # just extract the binary file from the tar.gz
    tar -C ${WORKSPACE}/bin -xpf ${gh_tar_full_path} ${gh_filename}/bin/gh --strip-components=2

    chmod +x ${WORKSPACE}/bin/gh
    rm -rf ${WORKSPACE}/tmp

    gh version
}

with_jq() {
    create_workspace "bin"
    retry 5 curl -sL -o ${WORKSPACE}/bin/jq "https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64"

    chmod +x ${WORKSPACE}/bin/jq
    jq --version
}

install_go_dependencies() {
    local install_packages=(
            "github.com/magefile/mage"
            "github.com/elastic/go-licenser"
            "golang.org/x/tools/cmd/goimports"
            "github.com/jstemmer/go-junit-report"
            "gotest.tools/gotestsum"
    )
    for pkg in "${install_packages[@]}"; do
        go install "${pkg}@latest"
    done
}