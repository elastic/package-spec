// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte(content), 0o644)
	require.NoError(t, err)
}

func writeDataStreamManifest(t *testing.T, dir, dsName, content string) {
	t.Helper()
	dsDir := filepath.Join(dir, "data_stream", dsName)
	err := os.MkdirAll(dsDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dsDir, "manifest.yml"), []byte(content), 0o644)
	require.NoError(t, err)
}

func TestValidatePolicyTemplateDatastreamCategories(t *testing.T) {
	cases := []struct {
		title        string
		setup        func(t *testing.T, dir string)
		expectedErrs []string
	}{
		{
			title: "categories match",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
      - network
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - network
  - observability
`)
			},
		},
		{
			title: "categories mismatch",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - observability
  - security
`)
			},
			expectedErrs: []string{`policy template "mytemplate"`, `"mylogs"`},
		},
		{
			title: "data stream manifest has no categories field",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
type: logs
streams:
  - input: logfile
`)
			},
		},
		{
			title: "policy template has no categories field",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - observability
`)
			},
		},
		{
			title: "multiple data streams — one matches, one does not",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - logs_ok
      - logs_bad
    categories:
      - observability
`)
				writeDataStreamManifest(t, dir, "logs_ok", `
title: OK Logs
categories:
  - observability
`)
				writeDataStreamManifest(t, dir, "logs_bad", `
title: Bad Logs
categories:
  - security
`)
			},
			expectedErrs: []string{`"logs_bad"`},
		},
		{
			title: "multiple policy templates — one mismatch",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
policy_templates:
  - name: template_a
    data_streams:
      - ds_a
    categories:
      - observability
  - name: template_b
    data_streams:
      - ds_b
    categories:
      - network
`)
				writeDataStreamManifest(t, dir, "ds_a", `
title: DS A
categories:
  - security
`)
				writeDataStreamManifest(t, dir, "ds_b", `
title: DS B
categories:
  - network
`)
			},
			expectedErrs: []string{`"template_a"`, `"ds_a"`},
		},
		{
			title: "non-integration packages are skipped",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: input
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
`)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)

			errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(dir))

			if len(tc.expectedErrs) == 0 {
				assert.Empty(t, errs)
			} else {
				require.Len(t, errs, 1)
				for _, expected := range tc.expectedErrs {
					assert.Contains(t, errs[0].Error(), expected)
				}
			}
		})
	}
}
