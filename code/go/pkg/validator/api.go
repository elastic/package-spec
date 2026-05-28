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

// WithWarningsAsErrors makes validation warnings count as errors when v is true.
func WithWarningsAsErrors(v bool) Option {
	return func(va *Validator) { va.warningsAsErrors = v }
}

// NewFromPath returns a Validator for the package rooted at packageRootPath.
//
// For ModeLegacy and ModeSource the filesystem honours linked (.link) files;
// for ModeBuild linked files are blocked (matching a built package artifact).
func NewFromPath(mode Mode, packageRootPath string, opts ...Option) (*Validator, error) {
	info, err := os.Stat(packageRootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid package path %q: %w", packageRootPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("invalid package path %q: not a directory", packageRootPath)
	}

	var fsys fs.FS
	if mode == ModeBuild {
		fsys = linkedfiles.NewBlockFS(os.DirFS(packageRootPath))
	} else {
		// ModeLegacy and ModeSource: resolve linked files transparently.
		fsys = linkedfiles.NewFS(packageRootPath, os.DirFS(packageRootPath))
	}
	return buildValidator(mode, packageRootPath, fsys, nil, opts), nil
}

// NewFromZip returns a Validator for the package stored in the zip file at zipPath.
// The zip must contain exactly one top-level directory holding the package tree,
// which is the format produced by elastic-package build.
//
// The returned Validator owns the underlying zip reader; calling Validate closes it.
// Do not call Validate more than once on a Validator created by NewFromZip.
func NewFromZip(mode Mode, zipPath string, opts ...Option) (_ *Validator, err error) {
	r, openErr := zip.OpenReader(zipPath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open zip file (%s): %w", zipPath, openErr)
	}
	// Close the reader on any error path; on success the Validator takes ownership.
	defer func() {
		if err != nil {
			r.Close()
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

	// Zip archives never contain live symlinks, so always block linked files.
	fsys := linkedfiles.NewBlockFS(subDir)
	return buildValidator(mode, zipPath, fsys, r, opts), nil
}

// NewFromFS returns a Validator for the package accessible through fsys at location.
//
// Unless fsys is already a *linkedfiles.FS it is wrapped with a BlockFS that
// rejects linked files — matching the behaviour of ValidateFromFS.
func NewFromFS(mode Mode, location string, fsys fs.FS, opts ...Option) (*Validator, error) {
	info, err := fs.Stat(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("invalid package filesystem at %q: %w", location, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("invalid package filesystem at %q: root is not a directory", location)
	}

	if _, ok := fsys.(*linkedfiles.FS); !ok {
		fsys = linkedfiles.NewBlockFS(fsys)
	}
	return buildValidator(mode, location, fsys, nil, opts), nil
}

// buildValidator is the shared internal constructor.
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
// Validate closes it on return, so Validate must not be called more than once
// on such a Validator.
func (v *Validator) Validate() error {
	if v.closer != nil {
		defer v.closer.Close()
	}

	pkg, err := packages.NewPackageFromFS(v.location, v.fsys)
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
