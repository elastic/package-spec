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

// NewFromPath returns a Validator for the package rooted at packageRootPath.
//
// For ModeLegacy and ModeSource the filesystem honours linked (.link) files;
// for ModeBuild linked files are blocked (matching a built package artifact).
func NewFromPath(mode Mode, packageRootPath string) (*Validator, error) {
	if !mode.Valid() {
		return nil, fmt.Errorf("invalid validation mode %q", mode)
	}

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
	return buildValidator(mode, packageRootPath, fsys, nil), nil
}

// NewFromZip returns a Validator for the package stored in the zip file at zipPath.
//
// Zip files always contain built packages — they are the output format produced
// by elastic-package build and consumed by the package registry and Fleet.
// Validation always runs in ModeBuild; source-only artifacts (_dev/, .link files,
// external: ecs references) are therefore rejected.
//
// NOTE: ModeBuild-specific validation rules are not yet implemented; ModeBuild
// and ModeLegacy currently produce identical rule sets. This is intentional —
// the mode is set here so that future PRs can attach build-only rules without
// changing the public API.
//
// The returned Validator owns the underlying zip reader; calling Validate closes it.
// Do not call Validate more than once on a Validator created by NewFromZip.
func NewFromZip(zipPath string) (_ *Validator, err error) {
	r, openErr := zip.OpenReader(zipPath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open zip file (%s): %w", zipPath, openErr)
	}
	// Close the reader on any error path; on success the Validator takes ownership.
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

	// Zip archives contain built packages; linked files are always blocked.
	fsys := linkedfiles.NewBlockFS(subDir)
	return buildValidator(ModeBuild, zipPath, fsys, r), nil
}

// NewFromFS returns a Validator for the package accessible through fsys at location.
//
// Linked-file handling depends on mode:
//   - ModeSource: if fsys is not already a *linkedfiles.FS it is wrapped with
//     linkedfiles.NewFS so .link files are resolved transparently.
//   - ModeLegacy: a pre-wrapped *linkedfiles.FS is preserved as-is (links
//     resolved); any other filesystem is wrapped with BlockFS. This matches the
//     behaviour of the deprecated ValidateFromFS.
//   - ModeBuild: BlockFS is always applied, even if fsys is already a
//     *linkedfiles.FS, so linked files are unconditionally rejected.
func NewFromFS(mode Mode, location string, fsys fs.FS) (*Validator, error) {
	if !mode.Valid() {
		return nil, fmt.Errorf("invalid validation mode %q", mode)
	}

	info, err := fs.Stat(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("invalid package filesystem at %q: %w", location, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("invalid package filesystem at %q: root is not a directory", location)
	}

	_, isLinkedFS := fsys.(*linkedfiles.FS)
	switch mode {
	case ModeSource:
		if !isLinkedFS {
			fsys = linkedfiles.NewFS(location, fsys)
		}
	case ModeBuild:
		// Always block, even a pre-wrapped *linkedfiles.FS.
		fsys = linkedfiles.NewBlockFS(fsys)
	default: // ModeLegacy
		// Preserve a pre-wrapped *linkedfiles.FS; block everything else.
		// Matches the old validateFromFS behaviour.
		if !isLinkedFS {
			fsys = linkedfiles.NewBlockFS(fsys)
		}
	}
	return buildValidator(mode, location, fsys, nil), nil
}

// buildValidator is the shared internal constructor.
func buildValidator(mode Mode, location string, fsys fs.FS, closer io.Closer) *Validator {
	v := &Validator{
		mode:             mode,
		location:         location,
		fsys:             fsys,
		warningsAsErrors: common.IsDefinedWarningsAsErrors(),
		closer:           closer,
	}

	return v
}

// Validate runs package validation and returns any errors encountered.
//
// If the Validator was created by NewFromZip it owns an open zip reader;
// Validate closes it on return, so Validate must not be called more than once
// on such a Validator.
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
