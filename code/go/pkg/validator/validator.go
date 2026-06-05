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

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/internal/packages"
	internalvalidator "github.com/elastic/package-spec/v3/code/go/internal/validator"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/common"
)

// Mode is the validation context that controls semantic rules and linked-file handling.
type Mode = internalvalidator.Mode

var (
	// ModeLegacy preserves the original validation behavior.
	ModeLegacy Mode = internalvalidator.ModeLegacy
	// ModeSource validates a checked-out source tree.
	ModeSource Mode = internalvalidator.ModeSource
	// ModeBuild validates a built package artifact.
	ModeBuild Mode = internalvalidator.ModeBuild
)

// Validator holds the configuration for a package validation run.
// Create one with NewValidator, then call ValidateFromPath, ValidateFromZip, or ValidateFromFS.
type Validator struct {
	mode             Mode
	warningsAsErrors bool
}

// Option configures a Validator.
type Option func(*Validator)

// WithWarningsAsErrors controls whether validation warnings are promoted to errors.
// When enabled is true, warnings are reported as errors regardless of the
// PACKAGE_SPEC_WARNINGS_AS_ERRORS environment variable. When enabled is false,
// warnings remain warnings even if the environment variable is set.
func WithWarningsAsErrors(enabled bool) Option {
	return func(v *Validator) { v.warningsAsErrors = enabled }
}

// NewValidator creates a Validator for the given mode and options.
func NewValidator(mode Mode, opts ...Option) (*Validator, error) {
	if !mode.Valid() {
		return nil, fmt.Errorf("invalid validation mode %q", mode)
	}
	v := &Validator{
		mode:             mode,
		warningsAsErrors: common.IsDefinedWarningsAsErrors(),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v, nil
}

// ValidateFromPath validates the package at path on disk.
func (v *Validator) ValidateFromPath(path string) error {
	fsys := os.DirFS(path)
	if v.mode == ModeBuild {
		fsys = linkedfiles.NewBlockFS(fsys)
	} else {
		fsys = linkedfiles.NewFS(path, fsys)
	}

	return v.validate(path, fsys)
}

// ValidateFromZip validates the package stored in a zip file.
// Zip files are supported in ModeLegacy and ModeBuild only.
func (v *Validator) ValidateFromZip(zipPath string) error {
	if v.mode != ModeLegacy && v.mode != ModeBuild {
		return errors.New("zip files are only supported in ModeLegacy or ModeBuild")
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file (%s): %w", zipPath, err)
	}
	defer r.Close()

	dirs, err := fs.ReadDir(r, ".")
	if err != nil {
		return fmt.Errorf("failed to read root directory in zip file (%s): %w", zipPath, err)
	}
	if len(dirs) != 1 {
		return fmt.Errorf("a single directory is expected in zip file, %d found", len(dirs))
	}

	subDir, err := fs.Sub(r, dirs[0].Name())
	if err != nil {
		return err
	}

	fsys := linkedfiles.NewBlockFS(subDir)
	return v.validate(zipPath, fsys)
}

// ValidateFromFS validates the package accessible through fsys at location.
func (v *Validator) ValidateFromFS(location string, fsys fs.FS) error {
	if v.mode == ModeLegacy {
		// If we are not explicitly using the linkedfiles.FS, we wrap fsys with
		// a linkedfiles.BlockFS to block the use of linked files.
		if _, ok := fsys.(*linkedfiles.FS); !ok {
			fsys = linkedfiles.NewBlockFS(fsys)
		}
	} else if _, ok := fsys.(*linkedfiles.FS); ok && v.mode == ModeBuild {
		return errors.New("linked files are not supported in ModeBuild")
	} else if _, ok := fsys.(*linkedfiles.BlockFS); ok && v.mode == ModeSource {
		return errors.New("block linked files are not supported in ModeSource")
	}

	return v.validate(location, fsys)
}

func (v *Validator) validate(location string, fsys fs.FS) error {
	pkg, err := packages.NewPackageFromFS(location, fsys)
	if err != nil {
		return err
	}
	if pkg.SpecVersion == nil {
		return errors.New("could not determine specification version for package")
	}

	spec, err := internalvalidator.NewSpec(*pkg.SpecVersion, v.mode)
	if err != nil {
		return err
	}
	spec.WarningsAsErrors = v.warningsAsErrors

	if errs := spec.ValidatePackage(*pkg); len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateFromPath is a convenience function that creates a new Validator in ModeLegacy and calls ValidateFromPath.
// Deprecated: Use NewValidator and ValidateFromPath instead.
func ValidateFromPath(path string) error {
	v, err := NewValidator(ModeLegacy)
	if err != nil {
		return err
	}
	return v.ValidateFromPath(path)
}

// ValidateFromZip is a convenience function that creates a new Validator in ModeLegacy and calls ValidateFromZip.
// Deprecated: Use NewValidator and ValidateFromZip instead.
func ValidateFromZip(zipPath string) error {
	v, err := NewValidator(ModeLegacy)
	if err != nil {
		return err
	}
	return v.ValidateFromZip(zipPath)
}

// ValidateFromFS is a convenience function that creates a new Validator in ModeLegacy and calls ValidateFromFS.
// Deprecated: Use NewValidator and ValidateFromFS instead.
func ValidateFromFS(location string, fsys fs.FS) error {
	v, err := NewValidator(ModeLegacy)
	if err != nil {
		return err
	}
	return v.ValidateFromFS(location, fsys)
}
