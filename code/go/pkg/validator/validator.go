// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Masterminds/semver/v3"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/internal/packages"
	"github.com/elastic/package-spec/v3/code/go/internal/validator"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/common"
)

type specFn func(semver.Version) (*validator.Spec, error)

// ValidateFromPath validates a package located at the given path using the legacy specification.
// This function preserves byte-for-byte identical behavior with existing validation workflows.
// Linked (.link) files are resolved transparently.
// Deprecated: Use ValidateFromSourcePath or ValidateFromBuildPath depending on the package type.
func ValidateFromPath(packageRootPath string) error {
	// We wrap the fs.FS with a linkedfiles.LinksFS to handle linked files.
	linksFS := linkedfiles.NewFS(packageRootPath, os.DirFS(packageRootPath))

	legacySpec := func(version semver.Version) (*validator.Spec, error) {
		return validator.NewLegacySpec(version)
	}

	return validateFromFS(packageRootPath, linksFS, legacySpec)
}

// ValidateFromBuildPath validates a built package located at the given path.
// This function uses the build specification, appropriate for packages produced by
// elastic-package build, distributed as zip files, or served by the package registry.
// Linked files (.link) are blocked; source-only artifacts are rejected.
func ValidateFromBuildPath(packageRootPath string) error {
	fs := os.DirFS(packageRootPath)

	buildSpec := func(version semver.Version) (*validator.Spec, error) {
		return validator.NewBuildSpec(version)
	}
	return validateFromFS(packageRootPath, fs, buildSpec)
}

// ValidateFromSourcePath validates a package source tree located at the given path.
// This function uses the source specification for checked-out source trees.
// Linked (.link) files are resolved transparently.
func ValidateFromSourcePath(packageRootPath string) error {
	// We wrap the fs.FS with a linkedfiles.LinksFS to handle linked files.
	linksFS := linkedfiles.NewFS(packageRootPath, os.DirFS(packageRootPath))

	sourceSpec := func(version semver.Version) (*validator.Spec, error) {
		return validator.NewSourceSpec(version)
	}

	return validateFromFS(packageRootPath, linksFS, sourceSpec)
}

// ValidateFromZip validates a package in zip file format.
// This function uses the build specification since zip files are by definition built packages.
// Linked files (.link) are blocked; source-only artifacts are rejected.
func ValidateFromZip(packagePath string) error {
	r, err := zip.OpenReader(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file (%s): %w", packagePath, err)
	}
	defer r.Close()

	dirs, err := fs.ReadDir(r, ".")
	if err != nil {
		return fmt.Errorf("failed to read root directory in zip file (%s): %w", packagePath, err)
	}
	if len(dirs) != 1 {
		return fmt.Errorf("a single directory is expected in zip file, %d found", len(dirs))
	}

	subDir, err := fs.Sub(r, dirs[0].Name())
	if err != nil {
		return err
	}

	buildSpec := func(version semver.Version) (*validator.Spec, error) {
		return validator.NewBuildSpec(version)
	}

	return validateFromFS(packagePath, subDir, buildSpec)
}

// validateFromFS validates a package against the appropriate specification and returns any errors.
// Package files are obtained through the given filesystem.
func validateFromFS(location string, fsys fs.FS, specFn specFn) error {
	// If we are not explicitly using the linkedfiles.FS, we wrap fsys with
	// a linkedfiles.BlockFS to block the use of linked files.
	if _, ok := fsys.(*linkedfiles.FS); !ok {
		fsys = linkedfiles.NewBlockFS(fsys)
	}
	pkg, err := packages.NewPackageFromFS(location, fsys)
	if err != nil {
		return err
	}

	if pkg.SpecVersion == nil {
		return errors.New("could not determine specification version for package")
	}

	spec, err := specFn(*pkg.SpecVersion)
	if err != nil {
		return err
	}
	spec.WarningsAsErrors = common.IsDefinedWarningsAsErrors()

	if errs := spec.ValidatePackage(*pkg); len(errs) > 0 {
		return errs
	}

	return nil
}
