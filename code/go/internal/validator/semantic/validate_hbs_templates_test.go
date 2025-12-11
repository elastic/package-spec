// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateTemplateDir(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.Empty(t, errs)

	})
	t.Run("valid handlebars file", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)
		hbsFilePath := path.Join(templateDir, "template.hbs")
		hbsContent := `{{#if condition}}Valid Handlebars{{/if}}`
		err = os.WriteFile(hbsFilePath, []byte(hbsContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.Empty(t, errs)
	})

	t.Run("valid handlebars file template with multiline", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)
		hbsFilePath := path.Join(templateDir, "template.hbs")
		hbsContent := `audit_rules: "# Session data audit rules\n-a always,exit -F arch=b64 -S execve,execveat -k exec\n-a always,exit -F arch=b64 -S exit_group\n-a always,exit -F arch=b64 -S setsid\n{{escape_multiline_string audit_rules}}"`
		err = os.WriteFile(hbsFilePath, []byte(hbsContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.Empty(t, errs)
	})
	t.Run("invalid handlebars file", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)
		hbsFilePath := path.Join(templateDir, "template.hbs")
		hbsContent := `{{#if condition}}Valid Handlebars`
		err = os.WriteFile(hbsFilePath, []byte(hbsContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.NotEmpty(t, errs)
		assert.Len(t, errs, 1)
	})
	t.Run("valid linked handlebars file", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		pkgDirLinked := path.Join(tmpDir, "linked")
		err = os.MkdirAll(pkgDirLinked, 0o755)
		require.NoError(t, err)
		linkedHbsFilePath := path.Join(pkgDirLinked, "linked_template.hbs")
		linkedHbsContent := `{{#if condition}}Valid Linked Handlebars{{/if}}`
		err = os.WriteFile(linkedHbsFilePath, []byte(linkedHbsContent), 0o644)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)
		hbsFilePath := path.Join(templateDir, "template.hbs.link")
		hbsContent := `../../../linked/linked_template.hbs`
		err = os.WriteFile(hbsFilePath, []byte(hbsContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.Empty(t, errs)

	})
	t.Run("invalid linked handlebars file", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := path.Join(tmpDir, "package")
		err := os.MkdirAll(pkgDir, 0o755)
		require.NoError(t, err)

		pkgDirLinked := path.Join(tmpDir, "linked")
		err = os.MkdirAll(pkgDirLinked, 0o755)
		require.NoError(t, err)
		linkedHbsFilePath := path.Join(pkgDirLinked, "linked_template.hbs")
		linkedHbsContent := `{{#if condition}}Valid Linked Handlebars`
		err = os.WriteFile(linkedHbsFilePath, []byte(linkedHbsContent), 0o644)
		require.NoError(t, err)

		templateDir := path.Join(pkgDir, "agent", "input")
		err = os.MkdirAll(templateDir, 0o755)
		require.NoError(t, err)
		hbsFilePath := path.Join(templateDir, "template.hbs.link")
		hbsContent := `../../../linked/linked_template.hbs`
		err = os.WriteFile(hbsFilePath, []byte(hbsContent), 0o644)
		require.NoError(t, err)

		fsys := fspath.DirFS(pkgDir)
		errs := validateTemplateDir(fsys, path.Join("agent", "input"))
		require.NotEmpty(t, errs)
		assert.Len(t, errs, 1)
	})
}
