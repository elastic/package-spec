#!/bin/bash

# Exit immediately on failure, or if an undefined variable is used.
set -eu

function compliance_test() {
	local stack_version=$1
	local spec_version=$2

cat <<EOF
      - label: "Elastic Stack ${stack_version} compliance with Spec ${spec_version}"
        command: ".buildkite/scripts/run-installer-compliance.sh ${stack_version} ${spec_version}"
        artifact_paths:
          - compliance/report-*.xml
        env:
          ELASTIC_PACKAGE_CHECK_UPDATE_DISABLED: "true"
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
compliance_test 9.1.0-SNAPSHOT 3.3.3
compliance_test 8.17.3 3.3.0
compliance_test 8.14.0 3.1.5
compliance_test 8.9.0 2.7.0

# Annotate junit results.
cat <<EOF
      - wait: ~
        continue_on_failure: true
      - label: ":junit: Annotate compliance test results"
        plugins:
          - junit-annotate#v2.5.0:
              artifacts: "compliance/report-*.xml"
              context: "compliance"
              report-skipped: true
              always-annotate: true
        agents:
          provider: "gcp"
EOF
