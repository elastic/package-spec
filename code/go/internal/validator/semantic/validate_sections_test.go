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

func TestValidateSectionsManifest(t *testing.T) {
	cases := []struct {
		title    string
		manifest string
		errors   []string
	}{
		{
			title: "valid: sections defined, vars reference them",
			manifest: `
sections:
  - name: auth_section
    title: Authentication
vars:
  - name: username
    section: auth_section
  - name: password
    section: auth_section
`,
		},
		{
			title: "valid: no sections and no section references",
			manifest: `
vars:
  - name: username
  - name: password
`,
		},
		{
			title: "valid: sections defined but no vars reference them",
			manifest: `
sections:
  - name: auth_section
    title: Authentication
vars:
  - name: username
`,
		},
		{
			title: "valid: some vars reference sections, others do not",
			manifest: `
sections:
  - name: auth_section
    title: Authentication
vars:
  - name: region
  - name: username
    section: auth_section
`,
		},
		{
			title: "invalid: var references undefined section",
			manifest: `
vars:
  - name: username
    section: auth_section
`,
			errors: []string{
				`file "manifest.yml" is invalid: var "username" references undefined section "auth_section" in package root`,
			},
		},
		{
			title: "invalid: duplicate section name",
			manifest: `
sections:
  - name: auth_section
    title: Authentication
  - name: auth_section
    title: Authentication (duplicate)
vars:
  - name: username
    section: auth_section
`,
			errors: []string{
				`file "manifest.yml" is invalid: duplicate section name "auth_section" in package root`,
			},
		},
		{
			title: "valid: sections scoped to policy template",
			manifest: `
policy_templates:
  - sections:
      - name: auth_section
        title: Authentication
    vars:
      - name: username
        section: auth_section
`,
		},
		{
			title: "invalid: policy template var references section not defined at that level",
			manifest: `
sections:
  - name: auth_section
    title: Authentication
policy_templates:
  - vars:
      - name: username
        section: auth_section
`,
			errors: []string{
				`file "manifest.yml" is invalid: var "username" references undefined section "auth_section" in policy template`,
			},
		},
		{
			title: "valid: sections scoped to input",
			manifest: `
policy_templates:
  - inputs:
      - sections:
          - name: auth_section
            title: Authentication
        vars:
          - name: username
            section: auth_section
`,
		},
		{
			title: "invalid: input var references section not defined at that level",
			manifest: `
policy_templates:
  - inputs:
      - vars:
          - name: username
            section: auth_section
`,
			errors: []string{
				`file "manifest.yml" is invalid: var "username" references undefined section "auth_section" in input`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			var manifest sectionsManifest
			err := yaml.Unmarshal([]byte(c.manifest), &manifest)
			require.NoError(t, err)

			errors := validateSectionsManifest("manifest.yml", manifest)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}

func TestValidateSectionsScope(t *testing.T) {
	cases := []struct {
		title    string
		sections []manifestSection
		vars     []sectionsVar
		errors   []string
	}{
		{
			title: "valid: all section references resolve",
			sections: []manifestSection{
				{Name: "auth_section"},
				{Name: "advanced_section"},
			},
			vars: []sectionsVar{
				{Name: "username", Section: "auth_section"},
				{Name: "timeout", Section: "advanced_section"},
				{Name: "region"},
			},
		},
		{
			title: "valid: no vars with section attributes",
			sections: []manifestSection{
				{Name: "auth_section"},
			},
			vars: []sectionsVar{
				{Name: "username"},
			},
		},
		{
			title:    "valid: empty sections and vars",
			sections: nil,
			vars:     nil,
		},
		{
			title:    "invalid: var references undefined section",
			sections: nil,
			vars: []sectionsVar{
				{Name: "username", Section: "missing_section"},
			},
			errors: []string{
				`file "test.yml" is invalid: var "username" references undefined section "missing_section" in package root`,
			},
		},
		{
			title: "invalid: duplicate section name",
			sections: []manifestSection{
				{Name: "auth_section"},
				{Name: "auth_section"},
			},
			vars: nil,
			errors: []string{
				`file "test.yml" is invalid: duplicate section name "auth_section" in package root`,
			},
		},
		{
			title: "invalid: multiple vars reference undefined sections",
			sections: []manifestSection{
				{Name: "auth_section"},
			},
			vars: []sectionsVar{
				{Name: "username", Section: "auth_section"},
				{Name: "api_key", Section: "missing_section"},
				{Name: "token", Section: "another_missing"},
			},
			errors: []string{
				`file "test.yml" is invalid: var "api_key" references undefined section "missing_section" in package root`,
				`file "test.yml" is invalid: var "token" references undefined section "another_missing" in package root`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errors := validateSectionsScope("test.yml", "package root", c.sections, c.vars)
			assert.Len(t, errors, len(c.errors))
			for _, err := range errors {
				assert.Contains(t, c.errors, err.Error())
			}
		})
	}
}
