// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDeploymentModes(t *testing.T) {
	cases := []struct {
		title        string
		manifestYAML string
		expectedErrs []string
	}{
		{
			title: "valid - inputs support all enabled deployment modes",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      default:
        enabled: true
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: httpjson
        deployment_modes: ['default', 'agentless']
      - type: filestream
        deployment_modes: ['default']
`,
			expectedErrs: nil,
		},
		{
			title: "valid - input with no deployment_modes supports all",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      default:
        enabled: true
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: httpjson
        # No deployment_modes specified - supports all
`,
			expectedErrs: nil,
		},
		{
			title: "invalid - default mode enabled but no input supports it",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: httpjson
        deployment_modes: ['agentless']
`,
			expectedErrs: []string{
				`policy template "test" enables deployment mode "default" but no input supports this mode`,
			},
		},
		{
			title: "invalid - agentless mode enabled but no input supports it",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: httpjson
        deployment_modes: ['default']
`,
			expectedErrs: []string{
				`policy template "test" enables deployment mode "agentless" but no input supports this mode`,
			},
		},
		{
			title: "invalid - both modes enabled but inputs support none",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: httpjson
        deployment_modes: []
`,
			expectedErrs: []string{
				`policy template "test" enables deployment mode "default" but no input supports this mode`,
				`policy template "test" enables deployment mode "agentless" but no input supports this mode`,
			},
		},
		{
			title: "valid - no deployment modes enabled",
			manifestYAML: `
policy_templates:
  - name: test
    inputs:
      - type: httpjson
        deployment_modes: ['default']
`,
			expectedErrs: nil,
		},
		{
			title: "valid - multiple policy templates",
			manifestYAML: `
policy_templates:
  - name: test1
    inputs:
      - type: httpjson
        deployment_modes: ['default']
  - name: test2
    deployment_modes:
      default:
        enabled: false
      agentless:
        enabled: true
        organization: elastic
        division: observability
        team: test
    inputs:
      - type: filestream
        deployment_modes: ['agentless']
`,
			expectedErrs: nil,
		},
		{
			title: "invalid - input specifies unsupported deployment mode",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      default:
        enabled: true
    inputs:
      - type: httpjson
        deployment_modes: ['agentless']  # agentless is disabled by default
`,
			expectedErrs: []string{
				`policy template "test" enables deployment mode "default" but no input supports this mode`,
				`input "httpjson" in policy template "test" specifies unsupported deployment mode "agentless"`,
			},
		},
		{
			title: "invalid - input specifies multiple unsupported deployment modes",
			manifestYAML: `
policy_templates:
  - name: test
    deployment_modes:
      default:
        enabled: false
    inputs:
      - type: httpjson
        deployment_modes: ['default', 'agentless']  # both disabled
`,
			expectedErrs: []string{
				`input "httpjson" in policy template "test" specifies unsupported deployment mode "default"`,
				`input "httpjson" in policy template "test" specifies unsupported deployment mode "agentless"`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			// Create a temporary directory and manifest file
			tempDir, err := os.MkdirTemp("", "test-deployment-modes")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			manifestPath := filepath.Join(tempDir, "manifest.yml")
			err = os.WriteFile(manifestPath, []byte(c.manifestYAML), 0644)
			require.NoError(t, err)

			fsys := fspath.DirFS(tempDir)
			errs := ValidateDeploymentModes(fsys)

			if len(c.expectedErrs) == 0 {
				assert.Empty(t, errs)
			} else {
				require.Len(t, errs, len(c.expectedErrs))
				for i, expectedErr := range c.expectedErrs {
					assert.Contains(t, errs[i].Error(), expectedErr)
				}
			}
		})
	}
}
