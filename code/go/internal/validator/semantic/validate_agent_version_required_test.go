// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/stretchr/testify/require"
)

func TestValidateMinimumAgentVersion(t *testing.T) {
	cases := []struct {
		title        string
		manifestYAML string
		expectedErr  error
	}{
		{
			title: "valid - agent.version condition is present",
			manifestYAML: `
name: test-package
version: 1.0.0
conditions:
  agent:
    version: "8.0.0"
`,
			expectedErr: nil,
		},
		{
			title: "invalid - agent.version condition is missing",
			manifestYAML: `
name: test-package
version: 1.0.0
conditions:
  some.other.condition: "value"
`,
			expectedErr: errAgentVersionConditionMissing,
		},
		{
			title: "invalid - conditions block is missing",
			manifestYAML: `
name: test-package
version: 1.0.0
`,
			expectedErr: errAgentVersionConditionMissing,
		},
		{
			title: "invalid - agent.version condition is not a string",
			manifestYAML: `
name: test-package
version: 1.0.0
conditions:
  agent.version:
    min: "8.0.0"
`,
			expectedErr: errAgentVersionIncorrectType,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tempDir := t.TempDir()

			manifestPath := filepath.Join(tempDir, "manifest.yml")
			err := os.WriteFile(manifestPath, []byte(c.manifestYAML), 0644)
			require.NoError(t, err)

			fsys := fspath.DirFS(tempDir)
			errs := ValidateMinimumAgentVersion(fsys)

			if c.expectedErr != nil {
				require.ErrorContains(t, errs, c.expectedErr.Error())
			} else {
				require.Empty(t, errs)
			}
		})
	}

}
