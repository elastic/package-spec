// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
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
    version: "^8.0.0"
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
			expectedErr: nil,
		},
		{
			title: "invalid - agent.version condition is not a string",
			manifestYAML: `
name: test-package
version: 1.0.0
conditions:
  agent.version:
    min: "^8.0.0"
`,
			expectedErr: errAgentVersionIncorrectType,
		},
		{
			title: "invalid - agent.version condition is not a constraint",
			manifestYAML: `
name: test-package
version: 1.0.0
conditions:
  agent.version: test
`,
			expectedErr: errInvalidAgentVersionCondition,
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
				require.Len(t, errs, 1)
				require.ErrorIs(t, errs[0], c.expectedErr)
			} else {
				require.Empty(t, errs)
			}
		})
	}

}
