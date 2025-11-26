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

func TestValidateUniqueFields(t *testing.T) {
	t.Run("unique fields across data streams", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(d, "data_stream", "foo", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "foo", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "foo", "fields", "more-fields.yml"), []byte(`
- name: field3
  type: keyword
- name: field4
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateUniqueFields(fspath.DirFS(d))
		require.Len(t, errs, 0)
	})
	t.Run("non-unique fields across data streams", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(d, "data_stream", "foo", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "foo", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "foo", "fields", "more-fields.yml"), []byte(`
- name: field3
  type: keyword
- name: field4
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateUniqueFields(fspath.DirFS(d))
		require.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "field \"field2\" is defined multiple times for data stream \"foo\", found in:")
	})
	t.Run("non-unique fields across transforms", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "elasticsearch", "transform", "foo", "fields"), 0o755)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(d, "elasticsearch", "transform", "bar", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "foo", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "foo", "fields", "more-fields.yml"), []byte(`
- name: field3
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "bar", "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateUniqueFields(fspath.DirFS(d))
		require.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "field \"field2\" is defined multiple times for transform \"foo\", found in:")
	})

	t.Run("non-unique fields in input packages", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "fields", "fields.yml"), []byte(`
- name: field1
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "fields", "more-fields.yml"), []byte(`
- name: field3
  type: keyword
- name: field2
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateUniqueFields(fspath.DirFS(d))
		require.Len(t, errs, 1)
		assert.ErrorContains(t, errs[0], "field \"field2\" is defined multiple times, found in:")
	})
}
