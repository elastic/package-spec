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

func TestValidateRequiredVarGroups(t *testing.T) {
	cases := []struct {
		title    string
		manifest string
		errors   []string
	}{
		{
			title: "good",
			manifest: `
vars:
  - name: user
  - name: password
  - name: api_key
policy_templates:
  - inputs:
    - required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
		},
		{
			title: "variable defined in policy",
			manifest: `
vars:
  - name: user
  - name: password
policy_templates:
  - vars:
    - name: api_key
    inputs:
    - required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
		},
		{
			title: "variable defined in input",
			manifest: `
vars:
  - name: user
  - name: password
policy_templates:
  - inputs:
    - vars:
        - name: api_key
      required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
		},
		{
			title: "missing variable",
			manifest: `
vars:
  - name: user
  - name: password
policy_templates:
  - inputs:
    - required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
			errors: []string{
				`file "manifest.yml" is invalid: required var "api_key" in optional group is not defined`,
			},
		},
		{
			title: "variable defined as required",
			manifest: `
vars:
  - name: user
  - name: password
  - name: api_key
    required: true
policy_templates:
  - inputs:
    - required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
			errors: []string{
				`file "manifest.yml" is invalid: required var "api_key" in optional group is defined as always required`,
			},
		},
		{
			title: "variable defined as required in policy",
			manifest: `
vars:
  - name: user
  - name: password
policy_templates:
  - vars:
    - name: api_key
      required: true
    inputs:
    - required_vars:
        user_password:
          - name: user
          - name: password
        api_key:
          - name: api_key
`,
			errors: []string{
				`file "manifest.yml" is invalid: required var "api_key" in optional group is defined as always required`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var manifest requiredVarsManifest
			err := yaml.Unmarshal([]byte(c.manifest), &manifest)
			require.NoError(t, err)

			errors := validateRequiredVarGroupsManifest("manifest.yml", manifest)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}

func TestValidateDataStreamRequiredVarGroups(t *testing.T) {
	rawPkgManifest := `
vars:
  - name: host
policy_templates:
  - name: logs
    inputs:
    - type: logfile
      vars:
        - name: credentials
`
	var pkgManifest requiredVarsManifest
	err := yaml.Unmarshal([]byte(rawPkgManifest), &pkgManifest)
	require.NoError(t, err)

	cases := []struct {
		title    string
		manifest string
		errors   []string
	}{
		{
			title: "good",
			manifest: `
streams:
  - vars:
    - name: user
    - name: password
    - name: api_key
    required_vars:
      user_password:
        - name: user
        - name: password
      api_key:
        - name: api_key
`,
		},
		{
			title: "variable defined in manifest",
			manifest: `
streams:
  - vars:
    - name: user
    - name: password
    - name: api_key
    required_vars:
      user_password:
        - name: user
        - name: password
      api_key:
        - name: api_key
      host:
        - name: host
`,
		},
		{
			title: "variable defined in manifest input",
			manifest: `
streams:
  - input: logfile
    vars:
    - name: user
    - name: password
    - name: api_key
    required_vars:
      user_password:
        - name: user
        - name: password
      api_key:
        - name: api_key
      credentials:
        - name: credentials
`,
		},
		{
			title: "missing variable",
			manifest: `
streams:
  - required_vars:
      user_password:
        - name: user
        - name: password
      api_key:
        - name: api_key
    vars:
    - name: user
    - name: password
`,
			errors: []string{
				`file "manifest.yml" is invalid: required var "api_key" in optional group is not defined`,
			},
		},
		{
			title: "variable defined as required",
			manifest: `
streams:
  - required_vars:
      user_password:
        - name: user
        - name: password
      api_key:
        - name: api_key
    vars:
      - name: user
      - name: password
      - name: api_key
        required: true
`,
			errors: []string{
				`file "manifest.yml" is invalid: required var "api_key" in optional group is defined as always required`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var manifest requiredVarsDataStreamManifest
			err := yaml.Unmarshal([]byte(c.manifest), &manifest)
			require.NoError(t, err)

			errors := validateDataStreamRequiredVarGroupsManifest("manifest.yml", manifest, pkgManifest)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}
