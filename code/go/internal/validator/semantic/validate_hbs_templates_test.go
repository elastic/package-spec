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
)

func TestValidateFile(t *testing.T) {

	t.Run("no handlebars files", func(t *testing.T) {
		err := validateFile("")
		assert.NoError(t, err)
	})

	t.Run("valid handlebars files", func(t *testing.T) {
		tmp := t.TempDir()

		filePath := filepath.Join(tmp, "template.yml.hbs")
		err := os.WriteFile(filePath, []byte("{{#if foo}}hello{{/if}}"), 0o644)
		require.NoError(t, err)

		errs := validateFile(filePath)
		assert.Empty(t, errs)
	})

	t.Run("invalid handlebars files", func(t *testing.T) {
		tmp := t.TempDir()

		filePath := filepath.Join(tmp, "bad.hbs")
		// Unclosed block should produce a parse error.
		err := os.WriteFile(filePath, []byte("{{#if foo}}no end"), 0o644)
		require.NoError(t, err)

		err = validateFile(filePath)
		require.Error(t, err)
	})

}
