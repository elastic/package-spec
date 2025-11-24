// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidatePipelineTags(t *testing.T) {
	testCases := []struct {
		name     string
		pipeline string
		errors   []string
	}{
		{
			name: "good",
			pipeline: `
processors:
  - set:
      tag: set_1
      field: key1
      value: value1
  - set:
      tag: set_2
      field: key2
      value: value2
      on_failure:
        - set:
            tag: onfail_1
            field: fail_key_1
            key: fail_value_1
`,
		},
		{
			name: "missing-tag",
			pipeline: `
processors:
  - set:
      field: key1
      value: value1
      on_failure:
        - set:
            tag: onfail_1
            field: fail_key_1
            key: fail_value_1
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 3 missing required tag (SVR00006)`,
			},
		},
		{
			name: "missing-tag-nested",
			pipeline: `
processors:
  - set:
      tag: set_1
      field: key1
      value: value1
      on_failure:
        - set:
            field: fail_key_1
            value: fail_value_1
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 8 missing required tag (SVR00006)`,
			},
		},
		{
			name: "duplicate-tag",
			pipeline: `
processors:
  - set:
      tag: set_1
      field: key1
      value: value1
  - set:
      tag: set_1
      field: key2
      value: value2
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 7 has duplicate tag value: "set_1"`,
			},
		},
		{
			name: "duplicate-nested-tag",
			pipeline: `
processors:
  - set:
      tag: set_1
      field: key1
      value: value1
      on_failure:
        - set:
            tag: set_1
            field: fail_key_1
            value: fail_value_1
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 3 has duplicate tag value: "set_1"`,
			},
		},
		{
			name: "invalid-tag-value",
			pipeline: `
processors:
  - set:
      tag: 1
      field: key1
      value: value1
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 3 has invalid tag value (SVR00006)`,
			},
		},
		{
			name: "empty-tag-value",
			pipeline: `
processors:
  - set:
      tag: ''
      field: key1
      value: value1
`,
			errors: []string{
				`file "default.yml" is invalid: set processor at line 3 has empty tag value (SVR00006)`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pipeline ingestPipeline
			err := yaml.Unmarshal([]byte(tc.pipeline), &pipeline)
			require.NoError(t, err)

			errors := validatePipelineTags(&pipeline, "default.yml")
			assert.Len(t, errors, len(tc.errors))
			for _, err := range errors {
				assert.Contains(t, tc.errors, err.Error())
			}
		})
	}
}
