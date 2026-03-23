// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePipelineOnFailure(t *testing.T) {
	testCases := []struct {
		name     string
		pipeline string
		errors   []string
	}{
		{
			name: "good-set",
			pipeline: `
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
  - set:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
`,
		},
		{
			name: "good-append",
			pipeline: `
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
  - append:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
`,
		},
		{
			name: "bad-event-kind-missing",
			pipeline: `
on_failure:
  - append:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
`,
			errors: []string{
				`file "default.yml" is invalid: pipeline on_failure handler must set event.kind to "pipeline_error" (SVR00008)`,
			},
		},
		{
			name: "bad-event-kind-wrong-value",
			pipeline: `
on_failure:
  - set:
      field: event.kind
      value: event
  - append:
      field: error.message
      value: >-
        Processor '{{{ _ingest.on_failure_processor_type }}}'
        with tag '{{{ _ingest.on_failure_processor_tag }}}'
        in pipeline '{{{ _ingest.pipeline }}}'
        failed with message '{{{ _ingest.on_failure_message }}}'
`,
			errors: []string{
				`file "default.yml" is invalid: pipeline on_failure handler must set event.kind to "pipeline_error" (SVR00008)`,
			},
		},
		{
			name: "bad-error-message-missing",
			pipeline: `
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
`,
			errors: []string{
				`file "default.yml" is invalid: pipeline on_failure handler must set error.message (SVR00009)`,
			},
		},
		{
			name: "bad-error-message-wrong-value",
			pipeline: `
on_failure:
  - set:
      field: event.kind
      value: pipeline_error
  - set:
      field: error.message
      value: Pipeline failed
`,
			errors: []string{
				`file "default.yml" is invalid: pipeline on_failure error.message must include "_ingest.on_failure_processor_type" (SVR00009)`,
				`file "default.yml" is invalid: pipeline on_failure error.message must include "_ingest.on_failure_processor_tag" (SVR00009)`,
				`file "default.yml" is invalid: pipeline on_failure error.message must include "_ingest.on_failure_message" (SVR00009)`,
				`file "default.yml" is invalid: pipeline on_failure error.message must include "_ingest.pipeline" (SVR00009)`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pipeline ingestPipeline
			err := yaml.Unmarshal([]byte(tc.pipeline), &pipeline)
			require.NoError(t, err)

			errors := validatePipelineOnFailure(&pipeline, "default.yml")
			assert.Len(t, errors, len(tc.errors))
			for _, err := range errors {
				assert.Contains(t, tc.errors, err.Error())
			}
		})
	}
}
