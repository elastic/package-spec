// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// testPackagesDir is the root of the fixture packages used in integration tests.
const testPackagesDir = "../../../../test/packages"

// createPackageZip creates a temporary zip file containing the package at
// packagePath under a single top-level directory (matching elastic-package
// build output). Returns the path to the zip.
func createPackageZip(t *testing.T, packagePath string) string {
	t.Helper()
	pkgName := filepath.Base(packagePath)

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, pkgName+".zip")

	f, err := os.Create(zipPath)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	err = filepath.WalkDir(packagePath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(packagePath, path)
		if err != nil {
			return err
		}
		// Zip entries must use forward slashes; prefix with the package
		// directory name so the zip contains exactly one top-level dir.
		entryName := filepath.ToSlash(filepath.Join(pkgName, rel))

		w, err := zw.Create(entryName)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	require.NoError(t, err)

	return zipPath
}

// -----------------------------------------------------------------------
// TestNewFromPath_RejectsInvalidMode
//
// NewFromPath must validate the mode eagerly and return an error at
// construction time, not defer the failure to Validate().
// -----------------------------------------------------------------------

func TestNewFromPath_RejectsInvalidMode(t *testing.T) {
	v, err := NewFromPath(Mode("invalid"), filepath.Join(testPackagesDir, "good"))
	require.Nil(t, v)
	require.ErrorContains(t, err, `invalid validation mode "invalid"`)
}

// -----------------------------------------------------------------------
// TestNewFromFS_RejectsInvalidMode
//
// Same contract as TestNewFromPath_RejectsInvalidMode for NewFromFS.
// -----------------------------------------------------------------------

func TestNewFromFS_RejectsInvalidMode(t *testing.T) {
	v, err := NewFromFS(Mode("invalid"), ".", os.DirFS("."))
	require.Nil(t, v)
	require.ErrorContains(t, err, `invalid validation mode "invalid"`)
}

// -----------------------------------------------------------------------
// TestNewFromZip_ConstructorSucceeds
//
// Zips up test/packages/good and verifies that NewFromZip constructs a
// Validator without error. NewFromZip always runs in ModeBuild; the good
// fixture is a source package so Validate() may return errors, but the
// constructor itself must succeed.
// -----------------------------------------------------------------------

func TestNewFromZip_ConstructorSucceeds(t *testing.T) {
	packagePath := filepath.Join(testPackagesDir, "good")
	zipPath := createPackageZip(t, packagePath)

	_, err := NewFromZip(zipPath)
	require.NoError(t, err)
}

// -----------------------------------------------------------------------
// TestNewFromFS_BuildModeBlocksLinksEvenWithPreWrappedLinkedFS
//
// ModeBuild must unconditionally block .link files, even when the caller
// passes a pre-wrapped *linkedfiles.FS that would otherwise resolve them.
// -----------------------------------------------------------------------

func TestNewFromFS_BuildModeBlocksLinksEvenWithPreWrappedLinkedFS(t *testing.T) {
	packagePath := filepath.Join(testPackagesDir, "with_links")
	linkedFS := linkedfiles.NewFS(packagePath, os.DirFS(packagePath))

	v, err := NewFromFS(ModeBuild, packagePath, linkedFS)
	require.NoError(t, err)

	errs := v.Validate()
	require.Error(t, errs)

	vErrs, ok := errs.(specerrors.ValidationErrors)
	require.True(t, ok, "expected ValidationErrors, got %T: %v", errs, errs)

	for _, e := range vErrs {
		if errors.Is(e, linkedfiles.ErrUnsupportedLinkFile) {
			return
		}
	}
	t.Fatal("expected at least one ErrUnsupportedLinkFile — ModeBuild must replace *linkedfiles.FS with BlockFS")
}

// -----------------------------------------------------------------------
// TestNewFromFS_LegacyAndSourceModePreservePreWrappedLinkedFS
//
// For ModeLegacy and ModeSource, a pre-wrapped *linkedfiles.FS must be
// preserved so .link files are resolved rather than blocked.
// ModeLegacy matches the old validateFromFS behaviour; ModeSource is the
// explicit source-tree contract.
// -----------------------------------------------------------------------

func TestNewFromFS_LegacyAndSourceModePreservePreWrappedLinkedFS(t *testing.T) {
	packagePath := filepath.Join(testPackagesDir, "with_links")

	for _, mode := range []Mode{ModeLegacy, ModeSource} {
		t.Run(string(mode), func(t *testing.T) {
			linkedFS := linkedfiles.NewFS(packagePath, os.DirFS(packagePath))
			v, err := NewFromFS(mode, packagePath, linkedFS)
			require.NoError(t, err)

			errs := v.Validate()
			if vErrs, ok := errs.(specerrors.ValidationErrors); ok {
				for _, e := range vErrs {
					if errors.Is(e, linkedfiles.ErrUnsupportedLinkFile) {
						t.Fatalf("%s with a pre-wrapped *linkedfiles.FS should resolve .link files, not block them", mode)
					}
				}
			}
		})
	}
}
