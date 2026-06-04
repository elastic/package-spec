// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/elastic/package-spec/v3/code/go/internal/linkedfiles"
	"github.com/elastic/package-spec/v3/code/go/internal/packages"
	internalvalidator "github.com/elastic/package-spec/v3/code/go/internal/validator"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/common"
)

// Validator holds the configuration for a package validation run.
// Create one with NewFromPath, NewFromZip, or NewFromFS, then call Validate.
type Validator struct {
	mode             Mode
	location         string
	fsys             fs.FS
	warningsAsErrors bool
	// closer is non-nil when the Validator owns a resource (e.g. a zip reader)
	// that must be released after Validate returns.
	closer io.Closer
}

// Option configures a Validator.
type Option func(*Validator)

// WithWarningsAsErrors makes validation warnings count as errors when enabled is true.
// This overrides the PACKAGE_SPEC_WARNINGS_AS_ERRORS environment variable.
func WithWarningsAsErrors(enabled bool) Option {
	return func(v *Validator) { v.warningsAsErrors = enabled }
}

// NewFromPath returns a Validator for the package rooted at packageRootPath.
// The mode determines which validation rules apply and how linked files are handled.
func NewFromPath(mode Mode, packageRootPath string, opts ...Option) (*Validator, error) {
	if !mode.internal.Valid() {
		return nil, fmt.Errorf("invalid validation mode %q", mode.internal)
	}
	fsys := mode.wrapFS(packageRootPath, os.DirFS(packageRootPath))
	return buildValidator(mode, packageRootPath, fsys, nil, opts), nil
}

// NewFromZip returns a Validator for the package stored in the zip file at zipPath.
//
// Zip files always contain built packages, so the validator always runs in
// ModeBuild; linked (.link) files are blocked.
//
// The returned Validator owns the underlying zip reader; Validate closes it.
// Do not call Validate more than once on a Validator created by NewFromZip.
func NewFromZip(zipPath string, opts ...Option) (_ *Validator, err error) {
	r, openErr := zip.OpenReader(zipPath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open zip file (%s): %w", zipPath, openErr)
	}
	defer func() {
		if err != nil {
			if cerr := r.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}
	}()

	dirs, err := fs.ReadDir(r, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory in zip file (%s): %w", zipPath, err)
	}
	if len(dirs) != 1 {
		return nil, fmt.Errorf("a single directory is expected in zip file, %d found", len(dirs))
	}

	subDir, err := fs.Sub(r, dirs[0].Name())
	if err != nil {
		return nil, err
	}

	fsys := linkedfiles.NewBlockFS(subDir)
	return buildValidator(ModeBuild, zipPath, fsys, r, opts), nil
}

// NewFromFS returns a Validator for the package accessible through fsys at location.
// The mode determines which validation rules apply; fsys is used as-is without any
// link-file wrapping. Callers are responsible for filesystem semantics.
func NewFromFS(mode Mode, location string, fsys fs.FS, opts ...Option) (*Validator, error) {
	if !mode.internal.Valid() {
		return nil, fmt.Errorf("invalid validation mode %q", mode.internal)
	}
	return buildValidator(mode, location, fsys, nil, opts), nil
}

func buildValidator(mode Mode, location string, fsys fs.FS, closer io.Closer, opts []Option) *Validator {
	v := &Validator{
		mode:             mode,
		location:         location,
		fsys:             fsys,
		warningsAsErrors: common.IsDefinedWarningsAsErrors(),
		closer:           closer,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate runs package validation and returns any errors encountered.
//
// If the Validator was created by NewFromZip it owns an open zip reader;
// Validate closes it on return, so Validate must not be called more than once.
func (v *Validator) Validate() (err error) {
	if v.closer != nil {
		defer func() {
			if cerr := v.closer.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}()
	}

	pkg, err := packages.NewPackageFromFS(v.location, v.fsys)
	if err != nil {
		return err
	}
	if pkg.SpecVersion == nil {
		return errors.New("could not determine specification version for package")
	}

	spec, err := internalvalidator.NewSpec(*pkg.SpecVersion, v.mode.internal)
	if err != nil {
		return err
	}
	spec.WarningsAsErrors = v.warningsAsErrors

	if errs := spec.ValidatePackage(*pkg); len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateFromPath validates a package located at the given path using the legacy specification.
// Linked (.link) files are resolved transparently.
// Deprecated: Use NewFromPath with ModeSource or ModeBuild depending on the package type.
func ValidateFromPath(packageRootPath string) error {
	v, err := NewFromPath(ModeLegacy, packageRootPath)
	if err != nil {
		return err
	}
	return v.Validate()
}

// ValidateFromZip validates a package in zip file format using the build specification.
// Linked files (.link) are blocked; zip files are by definition built packages.
func ValidateFromZip(packagePath string) error {
	v, err := NewFromZip(packagePath)
	if err != nil {
		return err
	}
	return v.Validate()
}
