// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"archive/zip"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// testPackagesDir is the root of the fixture packages used in integration tests.
const testPackagesDir = "../../../../test/packages"

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// assertErrorsEqual asserts that legacy and newAPI produce the same set of
// errors (both nil, or both non-nil with identical error messages).
//
// ValidationErrors are compared as unordered sets because the internal
// validator iterates Go maps whose iteration order is non-deterministic.
func assertErrorsEqual(t *testing.T, legacy, newAPI error) {
	t.Helper()
	switch {
	case legacy == nil && newAPI == nil:
		// both happy — pass
	case legacy == nil:
		t.Fatalf("legacy returned nil but new API returned: %v", newAPI)
	case newAPI == nil:
		t.Fatalf("new API returned nil but legacy returned: %v", legacy)
	default:
		assert.ElementsMatch(t,
			extractMessages(legacy),
			extractMessages(newAPI),
			"error sets differ between legacy and new API")
	}
}

// extractMessages unpacks a specerrors.ValidationErrors into individual
// message strings, or wraps a plain error in a single-element slice.
func extractMessages(err error) []string {
	if verrs, ok := err.(specerrors.ValidationErrors); ok {
		msgs := make([]string, len(verrs))
		for i, ve := range verrs {
			msgs[i] = ve.Error()
		}
		return msgs
	}
	return []string{err.Error()}
}

// listTestPackages returns the names of every directory under testPackagesDir.
func listTestPackages(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(testPackagesDir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

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
// TestLegacyPreservation_FromPath
//
// For every package under test/packages/, asserts that the new
// NewFromPath(ModeLegacy, ...).Validate() produces byte-for-byte identical
// output to the existing ValidateFromPath().
// -----------------------------------------------------------------------

func TestLegacyPreservation_FromPath(t *testing.T) {
	for _, pkgName := range listTestPackages(t) {
		t.Run(pkgName, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(testPackagesDir, pkgName)

			legacyErr := ValidateFromPath(path)
			newV, err := NewFromPath(ModeLegacy, path)
			require.NoError(t, err)
			newErr := newV.Validate()

			assertErrorsEqual(t, legacyErr, newErr)
		})
	}
}

// -----------------------------------------------------------------------
// TestLegacyPreservation_FromFS
//
// Parametrized over the same packages: ValidateFromFS with os.DirFS must
// match NewFromFS(ModeLegacy, ...) with the same filesystem.
// -----------------------------------------------------------------------

func TestLegacyPreservation_FromFS(t *testing.T) {
	for _, pkgName := range listTestPackages(t) {
		t.Run(pkgName, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(testPackagesDir, pkgName)

			// Use a fresh os.DirFS for each call to avoid any shared-state
			// side-effects (both calls are read-only, but being explicit is cleaner).
			legacyErr := ValidateFromFS(path, os.DirFS(path))
			newV, err := NewFromFS(ModeLegacy, path, os.DirFS(path))
			require.NoError(t, err)
			newErr := newV.Validate()

			assertErrorsEqual(t, legacyErr, newErr)
		})
	}
}

// -----------------------------------------------------------------------
// TestLegacyPreservation_FromZip (golden test)
//
// Zips up test/packages/good, runs both ValidateFromZip and
// NewFromZip(ModeLegacy,...).Validate(), and asserts identical output.
// This is the first zip-path coverage in this repository.
// -----------------------------------------------------------------------

func TestLegacyPreservation_FromZip(t *testing.T) {
	packagePath := filepath.Join(testPackagesDir, "good")
	zipPath := createPackageZip(t, packagePath)

	legacyErr := ValidateFromZip(zipPath)

	// ValidateFromZip closes the in-memory zip reader, not the file on disk,
	// so the same path can be reopened for the new-API call.
	newV, err := NewFromZip(ModeLegacy, zipPath)
	require.NoError(t, err)
	newErr := newV.Validate()

	assertErrorsEqual(t, legacyErr, newErr)
}
