#!/bin/bash
  
# Exit immediately on failure, or if an undefined variable is used.
set -eu

function compliance_test() {
	local stack_version=$1
	local spec_version=$2

cat <<EOF
      - label: "Elastic Stack ${stack_version} compliance with Spec ${spec_version}"
        command: ".buildkite/scripts/run-installer-compliance.sh ${stack_version} ${spec_version}"
        agents:
          provider: "gcp"
EOF
}
  
# Begin the pipeline.yml file.
cat <<EOF
steps:
  - group: ":terminal: Compliance test suites"
    key: "compliance-tests"
    steps:
EOF

# Generate each test we want to do.
compliance_test 8.10.0-SNAPSHOT 2.10.0
compliance_test 8.9.0 2.7.0
