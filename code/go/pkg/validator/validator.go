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
	"github.com/elastic/package-spec/v3/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
	// We wrap the fs.FS with a linkedfiles.LinksFS to handle linked files.
	linksFS := linkedfiles.NewFS(packageRootPath, os.DirFS(packageRootPath))
	return ValidateFromFS(packageRootPath, linksFS)
}

// ValidateFromZip validates a package on its zip format.
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

	return ValidateFromFS(packagePath, subDir)
}

// ValidateFromFS validates a package against the appropiate specification and returns any errors.
// Package files are obtained through the given filesystem.
func ValidateFromFS(location string, fsys fs.FS) error {
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

	spec, err := validator.NewSpec(*pkg.SpecVersion)
	if err != nil {
		return err
	}

	if errs := spec.ValidatePackage(*pkg); len(errs) > 0 {
		return errs
	}

	return nil
}
