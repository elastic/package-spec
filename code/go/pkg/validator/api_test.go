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
	"strings"
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

// applyPackageFilter applies the validation.yml filter from pkgPath if it exists,
// returning the filtered error (or the original if no filter is present).
//
// This mirrors the filter logic in TestValidateFile so that packages with known
// excluded error codes (e.g. SVR00006 in good_deployer_system_benchmark) are
// treated the same way across all integration tests.
func applyPackageFilter(t *testing.T, pkgPath string, err error) error {
	t.Helper()
	if err == nil {
		return nil
	}
	verrs, ok := err.(specerrors.ValidationErrors)
	if !ok {
		return err
	}
	filterConfig, filterErr := specerrors.LoadConfigFilter(os.DirFS(pkgPath))
	if errors.Is(filterErr, os.ErrNotExist) {
		return err // no validation.yml — return original
	}
	require.NoError(t, filterErr, "loading validation.yml for %s", pkgPath)
	filter := specerrors.NewFilter(filterConfig)
	result, filterRunErr := filter.Run(verrs)
	require.NoError(t, filterRunErr, "running filter for %s", pkgPath)
	return result.Processed
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
// TestSourceMode_GoodPackages
//
// Asserts that every good_* fixture under test/packages/ passes source-mode
// validation. Source mode runs all the rules that legacy mode runs plus
// source-only checks (e.g. rejecting _embedded_ecs in dynamic_templates).
// All packages prefixed "good_" are expected to represent valid source trees.
// -----------------------------------------------------------------------

func TestSourceMode_GoodPackages(t *testing.T) {
	for _, pkgName := range listTestPackages(t) {
		if pkgName != "good" && !strings.HasPrefix(pkgName, "good_") {
			continue
		}
		t.Run(pkgName, func(t *testing.T) {
			t.Parallel()
			pkgPath := filepath.Join(testPackagesDir, pkgName)

			v, err := NewFromPath(ModeSource, pkgPath)
			require.NoError(t, err)
			rawErr := v.Validate()
			filteredErr := applyPackageFilter(t, pkgPath, rawErr)
			assert.NoError(t, filteredErr, "package %q should be valid in source mode (after applying validation.yml filter)", pkgName)
		})
	}
}

// -----------------------------------------------------------------------
// TestSourceMode_BadEmbeddedEcs
//
// Asserts that a package containing _embedded_ecs keys in dynamic_templates:
//   - passes legacy validation  (_embedded_ecs is not checked in legacy mode)
//   - fails source-mode validation with a clear reference to the offending key
//   - passes build-mode validation  (rule is source-only, not build-only)
// -----------------------------------------------------------------------

func TestSourceMode_BadEmbeddedEcs(t *testing.T) {
	packagePath := filepath.Join(testPackagesDir, "bad_embedded_ecs")

	// Legacy mode: ValidateNoEmbeddedEcsInDynamicTemplates does not run → passes.
	legacyV, err := NewFromPath(ModeLegacy, packagePath)
	require.NoError(t, err)
	assert.NoError(t, legacyV.Validate(), "bad_embedded_ecs should pass legacy validation")

	// Source mode: _embedded_ecs is rejected → fails with a clear error.
	sourceV, err := NewFromPath(ModeSource, packagePath)
	require.NoError(t, err)
	sourceErr := sourceV.Validate()
	require.Error(t, sourceErr, "bad_embedded_ecs should fail source-mode validation")
	assert.Contains(t, sourceErr.Error(), "_embedded_ecs")

	// Build mode: rule is source-only (modes:[Source]), not build-only → passes.
	buildV, err := NewFromPath(ModeBuild, packagePath)
	require.NoError(t, err)
	assert.NoError(t, buildV.Validate(), "bad_embedded_ecs should pass build-mode validation (_embedded_ecs is a build artifact)")
}

// -----------------------------------------------------------------------
// TestBuildMode_SkipsBuildExcludedRules
//
// Verifies that rules tagged modes:[Legacy, Source] (i.e. excluded from build
// mode) do not fire when validating in ModeBuild.
//
// ValidateExternalFieldsWithDevFolder is the canary: good_integration_with_dev_tools
// has external fields and a _dev/build/build.yml; it passes in source mode via
// that rule — in build mode the rule must be absent (no "external key defined" error).
// -----------------------------------------------------------------------

func TestBuildMode_SkipsBuildExcludedRules(t *testing.T) {
	// good_integration_with_dev_tools has external fields + _dev/build/build.yml
	// and is the canonical fixture for ValidateExternalFieldsWithDevFolder.
	packagePath := filepath.Join(testPackagesDir, "good_integration_with_dev_tools")

	buildV, err := NewFromPath(ModeBuild, packagePath)
	require.NoError(t, err)

	buildErr := buildV.Validate()
	if buildErr != nil {
		// The only acceptable errors here are from build-mode rejection rules.
		// ValidateExternalFieldsWithDevFolder must NOT have contributed an error.
		assert.NotContains(t, buildErr.Error(), "external key defined",
			"ValidateExternalFieldsWithDevFolder must not run in build mode")
	}
}

// -----------------------------------------------------------------------
// TestBuildMode_NoDevFolder
//
// Verifies that ModeBuild:
//   - Accepts the canonical good_built fixture (no _dev/ present).
//   - Rejects a package containing a _dev/ directory with a clear error.
//   - Does NOT reject _dev/ when validating in ModeSource (source allows _dev/).
// -----------------------------------------------------------------------

func TestBuildMode_NoDevFolder(t *testing.T) {
	goodBuiltPath := filepath.Join(testPackagesDir, "build_mode", "good_built")
	badBuiltPath := filepath.Join(testPackagesDir, "build_mode", "bad_built_with_dev")

	t.Run("good_built passes ModeBuild", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, goodBuiltPath)
		require.NoError(t, err)
		assert.NoError(t, v.Validate(), "good_built should pass build-mode validation")
	})

	t.Run("bad_built_with_dev fails ModeBuild", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badBuiltPath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_with_dev should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), "_dev directory is not allowed in built packages")
	})

	t.Run("bad_built_with_dev passes ModeSource (source allows _dev/)", func(t *testing.T) {
		// bad_built_with_dev is not a valid source package (its _dev/build/build.yml
		// is empty and other source rules fire). We can only assert that the
		// build-only rule — ValidateNoDevFolder — did not contribute an error.
		v, err := NewFromPath(ModeSource, badBuiltPath)
		require.NoError(t, err)
		sourceErr := v.Validate()
		if sourceErr != nil {
			assert.NotContains(t, sourceErr.Error(), "_dev directory is not allowed",
				"ValidateNoDevFolder must not run in source mode")
		}
	})
}

// -----------------------------------------------------------------------
// TestBuildMode_NoLinkFiles
//
// Verifies that ModeBuild:
//   - Rejects a package containing a .link file with a clean error.
//   - Does NOT raise a .link error when validating in ModeSource (source
//     resolves .link files transparently via linkedfiles.NewFS).
// -----------------------------------------------------------------------

func TestBuildMode_NoLinkFiles(t *testing.T) {
	badBuiltPath := filepath.Join(testPackagesDir, "build_mode", "bad_built_with_link")
	withLinksPath := filepath.Join(testPackagesDir, "with_links")

	t.Run("bad_built_with_link fails ModeBuild", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badBuiltPath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_with_link should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), ".link files are not allowed in built packages")
	})

	t.Run("with_links passes ModeSource (.link files allowed in source mode)", func(t *testing.T) {
		// with_links resolves .link files transparently in source mode; other
		// source rules may still fire on that fixture. We can only assert that
		// ValidateNoLinkFiles did not contribute an error.
		v, err := NewFromPath(ModeSource, withLinksPath)
		require.NoError(t, err)
		sourceErr := v.Validate()
		if sourceErr != nil {
			assert.NotContains(t, sourceErr.Error(), ".link files are not allowed",
				"ValidateNoLinkFiles must not run in source mode")
		}
	})
}

// -----------------------------------------------------------------------
// TestBuildMode_NoExternalEcs
//
// Verifies that ModeBuild:
//   - Rejects a package containing a field with external: ecs.
//   - Does NOT raise an external-ecs error in ModeSource (source allows
//     external: ecs references via the build dependency mechanism).
// -----------------------------------------------------------------------

func TestBuildMode_NoExternalEcs(t *testing.T) {
	badBuiltPath := filepath.Join(testPackagesDir, "build_mode", "bad_built_external_ecs")
	goodV3Path := filepath.Join(testPackagesDir, "good_v3")

	t.Run("bad_built_external_ecs fails ModeBuild", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badBuiltPath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_external_ecs should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), "external: ecs")
		assert.Contains(t, buildErr.Error(), "ECS fields must be materialized")
	})

	t.Run("good_v3 with external: ecs passes ModeSource", func(t *testing.T) {
		// good_v3 is a known-good source fixture; it must pass source-mode cleanly.
		v, err := NewFromPath(ModeSource, goodV3Path)
		require.NoError(t, err)
		require.NoError(t, v.Validate(), "good_v3 should pass source-mode validation")
	})
}

// -----------------------------------------------------------------------
// TestBuildMode_StreamInputMaterialized
//
// Verifies that ModeBuild:
//   - Rejects a data stream manifest stream that carries a source-only 'package:' field.
//   - Rejects a data stream manifest stream that is missing the required 'input:' field.
//   - Rejects a package manifest policy_template input that carries a source-only 'package:' field.
//   - Does NOT raise these errors in ModeSource (source allows composable-input pattern).
//   - good_built still passes.
// -----------------------------------------------------------------------

func TestBuildMode_StreamInputMaterialized(t *testing.T) {
	goodBuiltPath := filepath.Join(testPackagesDir, "build_mode", "good_built")
	badStreamPackagePath := filepath.Join(testPackagesDir, "build_mode", "bad_built_stream_package")
	badMissingInputPath := filepath.Join(testPackagesDir, "build_mode", "bad_built_missing_input")
	badPolicyTemplatePkgPath := filepath.Join(testPackagesDir, "build_mode", "bad_built_policy_template_package")

	t.Run("good_built passes ModeBuild (stream materialization)", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, goodBuiltPath)
		require.NoError(t, err)
		assert.NoError(t, v.Validate(), "good_built should pass build-mode validation")
	})

	t.Run("bad_built_stream_package fails ModeBuild (package: in data stream)", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badStreamPackagePath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_stream_package should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), "'package:'")
		assert.Contains(t, buildErr.Error(), "source-only")
	})

	t.Run("bad_built_stream_package passes ModeSource (package: allowed in source)", func(t *testing.T) {
		// bad_built_stream_package is not a valid source package (it lacks a
		// _dev/build/build.yml for its external ECS field, among other issues).
		// We can only assert that ValidateStreamInputMaterialized did not fire.
		v, err := NewFromPath(ModeSource, badStreamPackagePath)
		require.NoError(t, err)
		sourceErr := v.Validate()
		if sourceErr != nil {
			assert.NotContains(t, sourceErr.Error(), "source-only",
				"ValidateStreamInputMaterialized must not run in source mode")
		}
	})

	t.Run("bad_built_missing_input fails ModeBuild (missing input: in data stream)", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badMissingInputPath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_missing_input should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), "missing required 'input:'")
	})

	t.Run("bad_built_policy_template_package fails ModeBuild (package: in policy_template input)", func(t *testing.T) {
		v, err := NewFromPath(ModeBuild, badPolicyTemplatePkgPath)
		require.NoError(t, err)
		buildErr := v.Validate()
		require.Error(t, buildErr, "bad_built_policy_template_package should fail build-mode validation")
		assert.Contains(t, buildErr.Error(), "'package:'")
		assert.Contains(t, buildErr.Error(), "source-only")
		assert.Contains(t, buildErr.Error(), "build packages must use 'type:'")
	})

	t.Run("bad_built_policy_template_package passes ModeSource (package: allowed in source)", func(t *testing.T) {
		// bad_built_policy_template_package has an unlisted package reference in its
		// requires section, so other source rules fire. We can only assert that
		// ValidateStreamInputMaterialized did not fire.
		v, err := NewFromPath(ModeSource, badPolicyTemplatePkgPath)
		require.NoError(t, err)
		sourceErr := v.Validate()
		if sourceErr != nil {
			assert.NotContains(t, sourceErr.Error(), "build packages must use 'type:'",
				"ValidateStreamInputMaterialized must not run in source mode")
		}
	})
}

// -----------------------------------------------------------------------
// TestNewFromZip_BuildMode
//
// NewFromZip always validates in ModeBuild — zip files are built packages.
// This test zips the canonical good_built fixture and confirms no errors.
// -----------------------------------------------------------------------

func TestNewFromZip_BuildMode(t *testing.T) {
	builtPath := filepath.Join(testPackagesDir, "build_mode", "good_built")
	zipPath := createPackageZip(t, builtPath)

	v, err := NewFromZip(zipPath)
	require.NoError(t, err)
	assert.NoError(t, v.Validate(), "good_built should pass ModeBuild validation via zip")
}
