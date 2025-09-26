// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInputPolicyTemplates(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fspath.FS
		wantErr bool
		errMsg  string
	}{
		{
			name: "input_manifest_with_policy_template_success",
			fsys: newMockFS().
				WithFile("manifest.yml", `
type: input
policy_templates:
  - name: udp
    template_path: udp.yml.hbs
`).
				WithFile("agent/input/udp.yml.hbs", "# UDP template"),
			wantErr: false,
		},
		{
			name: "input_manifest_with_policy_template_missing_template_file",
			fsys: newMockFS().
				WithFile("manifest.yml", `
type: input
policy_templates:
  - name: udp
    template_path: missing.yml.hbs
`),
			wantErr: true,
			errMsg:  "references template_path \"missing.yml.hbs\" but file",
		},
		{
			name: "integration_manifest_with_policy_template_success",
			fsys: newMockFS().
				WithFile("manifest.yml", `
type: integration
inputs:
  - title: Test UDP
    policy_templates:
      - name: Test UDP
        template_path: udp.yml.hbs
`).
				WithFile("agent/input/udp.yml.hbs", "# UDP template"),
			wantErr: false,
		},
		{
			name: "missing_template_file",
			fsys: newMockFS().
				WithFile("manifest.yml", `
type: integration
inputs:
  - title: Test UDP
    policy_templates:
      - name: Test UDP
        template_path: missing.yml.hbs
`),
			wantErr: true,
			errMsg:  "references template_path \"missing.yml.hbs\" but file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateInputPolicyTemplates(tt.fsys)

			if tt.wantErr {
				require.NotEmpty(t, errs, "expected validation errors")
				assert.Contains(t, errs.Error(), tt.errMsg)
			} else {
				assert.Empty(t, errs, "expected no validation errors")
			}
		})
	}
}
