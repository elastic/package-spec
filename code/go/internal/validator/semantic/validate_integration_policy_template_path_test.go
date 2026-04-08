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

func writeMinimalIntegrationManifest(t *testing.T, root string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(root, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: pt
    title: t
    description: d
    inputs:
      - type: nginx/access
        title: a
        description: b
`), 0o644)
	require.NoError(t, err)
}

func TestValidateIntegrationPolicyTemplates_DataStreamStreams(t *testing.T) {
	t.Run("all stream templates valid", func(t *testing.T) {
		d := t.TempDir()
		writeMinimalIntegrationManifest(t, d)
		err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "access.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
    title: A
    description: d
    template_path: access.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs)
	})

	t.Run("missing stream template_path file", func(t *testing.T) {
		d := t.TempDir()
		writeMinimalIntegrationManifest(t, d)
		err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/access
    title: A
    description: d
    template_path: access.yml.hbs
  - input: nginx/error
    title: E
    description: d
    template_path: error_stream.yml.hbs
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "access.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), `data stream "data_stream/logs" stream input "nginx/error": template file not found`)
	})

	t.Run("default stream.yml.hbs when template_path omitted", func(t *testing.T) {
		d := t.TempDir()
		writeMinimalIntegrationManifest(t, d)
		err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "stream.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: nginx/other
    title: O
    description: d
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs)
	})

	t.Run("prefixed stream template filename", func(t *testing.T) {
		d := t.TempDir()
		writeMinimalIntegrationManifest(t, d)
		err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "prefixstream.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: prefix/stream
    title: P
    description: d
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs)
	})

	t.Run("template_paths on stream Fleet precedence over template_path", func(t *testing.T) {
		d := t.TempDir()
		writeMinimalIntegrationManifest(t, d)
		err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "a.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "b.yml.hbs"), []byte(`y`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: custom
    title: C
    description: d
    template_path: missing-only-this-would-matter.yml.hbs
    template_paths:
      - a.yml.hbs
      - b.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs)
	})

	t.Run("policy input template_path set still validates stream templates", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "input.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: pt
    title: t
    description: d
    inputs:
      - type: logfile
        title: a
        description: b
        template_path: input.yml.hbs
`), 0o644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: logfile
    title: L
    description: d
    template_path: does-not-exist.yml.hbs
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), `data stream "data_stream/logs" stream input "logfile": template file not found`)
	})

	t.Run("policy input template_paths", func(t *testing.T) {
		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "agent", "input"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "t1.yml.hbs"), []byte(`x`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "agent", "input", "t2.yml.hbs"), []byte(`y`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "manifest.yml"), []byte(`
type: integration
policy_templates:
  - name: pt
    title: t
    description: d
    inputs:
      - type: logfile
        title: a
        description: b
        template_paths:
          - t1.yml.hbs
          - t2.yml.hbs
`), 0o644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", "stream.yml.hbs"), []byte(`z`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "manifest.yml"), []byte(`
streams:
  - input: logfile
    title: L
    description: d
`), 0o644)
		require.NoError(t, err)

		errs := ValidateIntegrationPolicyTemplates(fspath.DirFS(d))
		require.Empty(t, errs)
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
func TestFindPathAtDirectory(t *testing.T) {
	d := t.TempDir()

	dsDir := filepath.ToSlash(path.Join("data_stream", "logs", "agent", "stream"))
	err := os.MkdirAll(filepath.Join(d, "data_stream", "logs", "agent", "stream"), 0o755)
	require.NoError(t, err)

	t.Run("exact match", func(t *testing.T) {
		templatePath := "exact.yml.hbs"
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", templatePath), []byte("content"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", templatePath))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, templatePath)), foundFile)
	})

	t.Run("match with .link extension", func(t *testing.T) {
		templatePath := "linked.yml.hbs"
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", templatePath+".link"), []byte("content"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", templatePath+".link"))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, templatePath+".link")), foundFile)
	})

	t.Run("match with prefix", func(t *testing.T) {
		templatePath := "stream.yml.hbs"
		prefixedFile := "prefixstream.yml.hbs"
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile), []byte("content"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, prefixedFile)), foundFile)
	})

	t.Run("match with prefix and .link extension", func(t *testing.T) {
		templatePath := "stream.yml.hbs"
		prefixedFile := "prefixstream.yml.hbs.link"
		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile), []byte("content"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, prefixedFile)), foundFile)
	})

	t.Run("no match found", func(t *testing.T) {
		templatePath := "nonexistent.yml.hbs"

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.Empty(t, foundFile)
	})

	t.Run("multiple matches - exact match takes precedence", func(t *testing.T) {
		templatePath := "multi.yml.hbs"
		exactFile := "multi.yml.hbs"
		prefixedFile := "prefixmulti.yml.hbs"

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", exactFile), []byte("exact"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", exactFile))

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile), []byte("prefixed"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", prefixedFile))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, exactFile)), foundFile)
	})

	t.Run("link file takes precedence over suffix match", func(t *testing.T) {
		templatePath := "link.yml.hbs"
		linkFile := "link.yml.hbs.link"
		suffixFile := "prefixlink.yml.hbs"

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", linkFile), []byte("link"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", linkFile))

		err = os.WriteFile(filepath.Join(d, "data_stream", "logs", "agent", "stream", suffixFile), []byte("suffix"), 0o644)
		require.NoError(t, err)
		defer os.Remove(filepath.Join(d, "data_stream", "logs", "agent", "stream", suffixFile))

		foundFile, err := findPathAtDirectory(fspath.DirFS(d), dsDir, templatePath)
		require.NoError(t, err)
		require.NotEmpty(t, foundFile)
		require.Equal(t, filepath.ToSlash(path.Join(dsDir, linkFile)), foundFile)
	})
}
