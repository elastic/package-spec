// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestReadDataStreamsManifests(t *testing.T) {

	d := t.TempDir()

	err := os.MkdirAll(filepath.Join(d, "data_stream", "logs"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
    template_path: stream.yml.hbs
  - input: nginx/error
    template_path: error_stream.yml.hbs
`), 0o644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(d, "data_stream", "logs", "nested"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "nested", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
    template_path: stream.yml.hbs
  - input: nginx/error
    template_path: error_stream.yml.hbs
`), 0o644)
	require.NoError(t, err)

	dataStreamsManifestMap, err := readDataStreamsManifests(fspath.DirFS(d))
	require.NoError(t, err)
	// only the top-level manifest.yml should be read
	require.Len(t, dataStreamsManifestMap, 1)

	mapKey := filepath.ToSlash(path.Join("data_stream", "logs"))
	require.NotEmpty(t, dataStreamsManifestMap[mapKey])
	logsManifest := dataStreamsManifestMap[mapKey]
	require.Len(t, logsManifest.Streams, 2)
	require.Equal(t, "nginx/access", logsManifest.Streams[0].Input)
	require.Equal(t, "stream.yml.hbs", logsManifest.Streams[0].TemplatePath)
	require.Equal(t, "nginx/error", logsManifest.Streams[1].Input)
	require.Equal(t, "error_stream.yml.hbs", logsManifest.Streams[1].TemplatePath)
}

func TestValidateInputWithStreams(t *testing.T) {
	d := t.TempDir()
	err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "access.yml.hbs"), []byte(`access stream template`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "log.yml.hbs"), []byte(`default stream template`), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "syslog.yml.hbs"), []byte(`default stream template`), 0o644)
	require.NoError(t, err)

	dsMap := make(map[string]dataStreamManifest)
	dsMap[filepath.ToSlash(path.Join("data_stream", "logs"))] = dataStreamManifest{
		Streams: []stream{
			{
				Input:        "nginx/access",
				TemplatePath: "access.yml.hbs",
				// access.yml.hbs exists
			},
			{
				Input:        "nginx/error",
				TemplatePath: "error.yml.hbs",
				// error.yml.hbs does not exist
			},
			{
				Input: "nginx/default",
				// no template_path set, should default to stream.yml.hbs
			},
			{
				Input: "nginx/multiple",
				// exact match will not be found, but multiple files ending with log.yml.hbs and syslog.yml.hbs will be found
				TemplatePath: "og.yml.hbs",
			},
		},
	}
	t.Run("valid input with existing template_path", func(t *testing.T) {
		err = validateInputWithStreams(fspath.DirFS(d), "nginx/access", dsMap)
		require.NoError(t, err)
	})

	t.Run("input with non-existing template_path", func(t *testing.T) {
		err = validateInputWithStreams(fspath.DirFS(d), "nginx/error", dsMap)
		require.ErrorIs(t, errTemplateNotFound, err)
	})

	t.Run("valid input with default template_path", func(t *testing.T) {
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "stream.yml.hbs"), []byte(`default stream template`), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", "stream.yml.hbs"))
		err = validateInputWithStreams(fspath.DirFS(d), "nginx/other", dsMap)
		require.NoError(t, err)
	})

	t.Run("multiple templates found for input", func(t *testing.T) {
		err = validateInputWithStreams(fspath.DirFS(d), "nginx/multiple", dsMap)
		require.ErrorIs(t, err, errMultipleTemplatesFound)
	})

	t.Run("valid input with prefix default template_path", func(t *testing.T) {
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "filestream.yml.hbs"), []byte(`default stream template`), 0o644)
		require.NoError(t, err)

		err = validateInputWithStreams(fspath.DirFS(d), "nginx/default", dsMap)
		require.NoError(t, err)
	})

}
func TestValidateIntegrationPolicyTemplates_NonIntegrationType(t *testing.T) {
	d := t.TempDir()
	// write a manifest with a non-integration type
	err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`type: input`), 0o644)
	require.NoError(t, err)

	errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
	require.Nil(t, errs)
}

func TestValidateIntegrationPolicyTemplates_IntegrationValidTemplates(t *testing.T) {
	d := t.TempDir()

	// manifest: integration with a policy template referencing nginx/access (no template_path at policy level)
	err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: pt1
    inputs:
      - type: nginx/access
`), 0o644)
	require.NoError(t, err)

	// data stream manifest providing the stream for nginx/access with a specific template
	err = os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
    template_path: access.yml.hbs
`), 0o644)
	require.NoError(t, err)
	// write the actual template file referenced by the stream
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "access.yml.hbs"), []byte("template"), 0o644)
	require.NoError(t, err)

	errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
	require.Empty(t, errs)
}

func TestValidateIntegrationPolicyTemplates_DefaultTemplate(t *testing.T) {
	d := t.TempDir()

	// manifest: integration with a policy template referencing an input that does not exist in any data stream
	err := os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: pt2
    inputs:
    - type: nginx/access
`), 0o644)
	require.NoError(t, err)

	// create a data stream that does NOT include the referenced input
	err = os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
`), 0o644)
	require.NoError(t, err)
	// write the default template file for the existing stream
	err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "stream.yml.hbs"), []byte("template"), 0o644)
	require.NoError(t, err)

	errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
	require.Empty(t, errs)
}
