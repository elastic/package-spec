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

func TestValidateVarGroupsManifest(t *testing.T) {
	cases := []struct {
		title    string
		manifest string
		errors   []string
	}{
		{
			title: "valid var_groups",
			manifest: `
vars:
  - name: access_key_id
  - name: secret_access_key
  - name: role_arn
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
          - secret_access_key
      - name: assume_role
        vars:
          - role_arn
`,
		},
		{
			title: "variable in policy template",
			manifest: `
vars:
  - name: access_key_id
policy_templates:
  - vars:
    - name: secret_access_key
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
          - secret_access_key
`,
		},
		{
			title: "variable in input",
			manifest: `
vars:
  - name: access_key_id
policy_templates:
  - inputs:
    - vars:
      - name: secret_access_key
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
          - secret_access_key
`,
		},
		{
			title: "missing variable",
			manifest: `
vars:
  - name: access_key_id
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
          - secret_access_key
`,
			errors: []string{
				`file "manifest.yml" is invalid: var "secret_access_key" referenced in var_group "credential_type" option "direct_access_key" is not defined`,
			},
		},
		{
			title: "duplicate var_group name",
			manifest: `
vars:
  - name: access_key_id
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
  - name: credential_type
    options:
      - name: another_option
        vars:
          - access_key_id
`,
			errors: []string{
				`file "manifest.yml" is invalid: duplicate var_group name "credential_type"`,
			},
		},
		{
			title: "duplicate option name",
			manifest: `
vars:
  - name: access_key_id
  - name: secret_access_key
var_groups:
  - name: credential_type
    options:
      - name: direct_access_key
        vars:
          - access_key_id
      - name: direct_access_key
        vars:
          - secret_access_key
`,
			errors: []string{
				`file "manifest.yml" is invalid: duplicate option name "direct_access_key" in var_group "credential_type"`,
			},
		},
		{
			title: "no var_groups is valid",
			manifest: `
vars:
  - name: access_key_id
`,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var manifest varGroupsManifest
			err := yaml.Unmarshal([]byte(c.manifest), &manifest)
			require.NoError(t, err)

			errors := validateVarGroupsManifest("manifest.yml", manifest)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}

func TestValidateVarGroups(t *testing.T) {
	cases := []struct {
		title         string
		varGroups     []varGroup
		availableVars []string
		errors        []string
	}{
		{
			title: "valid - all vars exist",
			varGroups: []varGroup{
				{
					Name: "auth_type",
					Options: []varGroupOption{
						{Name: "basic", Vars: []string{"username", "password"}},
						{Name: "api_key", Vars: []string{"api_key"}},
					},
				},
			},
			availableVars: []string{"username", "password", "api_key"},
			errors:        nil,
		},
		{
			title: "missing var reference",
			varGroups: []varGroup{
				{
					Name: "auth_type",
					Options: []varGroupOption{
						{Name: "basic", Vars: []string{"username", "password", "missing_var"}},
					},
				},
			},
			availableVars: []string{"username", "password"},
			errors: []string{
				`file "test.yml" is invalid: var "missing_var" referenced in var_group "auth_type" option "basic" is not defined`,
			},
		},
		{
			title: "duplicate var_group names",
			varGroups: []varGroup{
				{Name: "auth_type", Options: []varGroupOption{{Name: "opt1", Vars: []string{}}}},
				{Name: "auth_type", Options: []varGroupOption{{Name: "opt2", Vars: []string{}}}},
			},
			availableVars: []string{},
			errors: []string{
				`file "test.yml" is invalid: duplicate var_group name "auth_type"`,
			},
		},
		{
			title: "duplicate option names within var_group",
			varGroups: []varGroup{
				{
					Name: "auth_type",
					Options: []varGroupOption{
						{Name: "basic", Vars: []string{}},
						{Name: "basic", Vars: []string{}},
					},
				},
			},
			availableVars: []string{},
			errors: []string{
				`file "test.yml" is invalid: duplicate option name "basic" in var_group "auth_type"`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errors := validateVarGroups("test.yml", c.varGroups, c.availableVars)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}
