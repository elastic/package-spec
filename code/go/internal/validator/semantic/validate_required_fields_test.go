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

func TestValidateRequiredFields(t *testing.T) {
	t.Run("all required fields present with correct types", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: constant_keyword
- name: data_stream.dataset
  type: constant_keyword
- name: data_stream.namespace
  type: constant_keyword
- name: "@timestamp"
  type: date
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 0)
	})
	t.Run("missing required fields", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: constant_keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 3)
	})
	t.Run("required fields with incorrect types", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "data_stream", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "data_stream", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: keyword
- name: data_stream.dataset
  type: text
- name: data_stream.namespace
  type: keyword
- name: "@timestamp"
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 4)
	})

	t.Run("all required fields present with correct types in transform", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "elasticsearch", "transform", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: constant_keyword
- name: data_stream.dataset
  type: constant_keyword
- name: data_stream.namespace
  type: constant_keyword
- name: "@timestamp"
  type: date
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 0)
	})
	t.Run("missing required fields in transform", func(t *testing.T) {

		d := t.TempDir()
		err := os.MkdirAll(filepath.Join(d, "elasticsearch", "transform", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: constant_keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 3)
	})
	t.Run("required fields with incorrect types in transform", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "elasticsearch", "transform", "test", "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "elasticsearch", "transform", "test", "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: keyword
- name: data_stream.dataset
  type: keyword
- name: data_stream.namespace
  type: keyword
- name: "@timestamp"
  type: keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		// should data_stream.type, data_stream.dataset, data_stream.namespace fields be enforced as constant_keyword too?
		require.Len(t, errs, 1)
	})

	t.Run("missing required fields in input package", func(t *testing.T) {
		d := t.TempDir()

		err := os.MkdirAll(filepath.Join(d, "fields"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "fields", "fields.yml"), []byte(`
- name: data_stream.type
  type: constant_keyword
`), 0o644)
		require.NoError(t, err)

		errs := ValidateRequiredFields(fspath.DirFS(d))
		require.Len(t, errs, 3)
	})
}
