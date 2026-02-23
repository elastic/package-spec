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

func TestValidatePolicyTemplateDatastreamCategories_Match(t *testing.T) {
	d := t.TempDir()

	writeManifest(t, d, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
      - network
`)
	writeDataStreamManifest(t, d, "mylogs", `
title: My Logs
categories:
  - network
  - observability
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Empty(t, errs)
}

func TestValidatePolicyTemplateDatastreamCategories_Mismatch(t *testing.T) {
	d := t.TempDir()

	writeManifest(t, d, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
	writeDataStreamManifest(t, d, "mylogs", `
title: My Logs
categories:
  - observability
  - security
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), `policy template "mytemplate"`)
	assert.Contains(t, errs[0].Error(), `"mylogs"`)
}

func TestValidatePolicyTemplateDatastreamCategories_NoDataStreamCategories(t *testing.T) {
	// Data stream manifest without categories field → should pass (nothing to validate)
	d := t.TempDir()

	writeManifest(t, d, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
	writeDataStreamManifest(t, d, "mylogs", `
title: My Logs
type: logs
streams:
  - input: logfile
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Empty(t, errs)
}

func TestValidatePolicyTemplateDatastreamCategories_NoPolicyTemplateCategories(t *testing.T) {
	// Policy template without categories field → should pass (nothing to validate)
	d := t.TempDir()

	writeManifest(t, d, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
`)
	writeDataStreamManifest(t, d, "mylogs", `
title: My Logs
categories:
  - observability
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Empty(t, errs)
}

func TestValidatePolicyTemplateDatastreamCategories_MultipleDataStreams(t *testing.T) {
	// Multiple data streams — one matches, one doesn't
	d := t.TempDir()

	writeManifest(t, d, `
type: integration
policy_templates:
  - name: mytemplate
    data_streams:
      - logs_ok
      - logs_bad
    categories:
      - observability
`)
	writeDataStreamManifest(t, d, "logs_ok", `
title: OK Logs
categories:
  - observability
`)
	writeDataStreamManifest(t, d, "logs_bad", `
title: Bad Logs
categories:
  - security
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), `"logs_bad"`)
}

func TestValidatePolicyTemplateDatastreamCategories_MultiplePolicyTemplates(t *testing.T) {
	// Two policy templates — each has one mismatch
	d := t.TempDir()

	writeManifest(t, d, `
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
	writeDataStreamManifest(t, d, "ds_a", `
title: DS A
categories:
  - security
`)
	writeDataStreamManifest(t, d, "ds_b", `
title: DS B
categories:
  - network
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), `"template_a"`)
	assert.Contains(t, errs[0].Error(), `"ds_a"`)
}

func TestValidatePolicyTemplateDatastreamCategories_NonIntegrationPackage(t *testing.T) {
	// Non-integration packages are skipped
	d := t.TempDir()

	writeManifest(t, d, `
type: input
policy_templates:
  - name: mytemplate
    data_streams:
      - mylogs
    categories:
      - observability
`)
	writeDataStreamManifest(t, d, "mylogs", `
title: My Logs
categories:
  - security
`)

	errs := ValidatePolicyTemplateDatastreamCategories(fspath.DirFS(d))
	assert.Empty(t, errs)
}
