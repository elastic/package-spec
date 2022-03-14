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

	"github.com/elastic/package-spec/code/go/internal/validator"
)

// ValidateFromPath validates a package located at the given path against the
// appropriate specification and returns any errors.
func ValidateFromPath(packageRootPath string) error {
	return ValidateFromFS(packageRootPath, os.DirFS(packageRootPath))
}

// ValidateFromZipReader validates a zip package obtained through a reader.
func ValidateFromZipReader(location string, reader io.ReaderAt, size int64) error {
	r, err := zip.NewReader(reader, size)
	if err != nil {
		return err
	}

	return validateZipReader(location, r)
}

// ValidateFromZip validates a package on its zip format.
func ValidateFromZip(packagePath string) error {
	r, err := zip.OpenReader(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file (%s): %w", packagePath, err)
	}
	defer r.Close()

	return validateZipReader(packagePath, &r.Reader)
}

func validateZipReader(location string, r *zip.Reader) error {
	dirs, err := fs.ReadDir(r, ".")
	if err != nil {
		return fmt.Errorf("failed to read root directory in zip file (%s): %w", location, err)
	}
	if len(dirs) != 1 {
		return fmt.Errorf("a single directory is expected in zip file, %d found", len(dirs))
	}

	subDir, err := fs.Sub(r, dirs[0].Name())
	if err != nil {
		return err
	}

	return ValidateFromFS(location, subDir)
}

// ValidateFromFS validates a package against the appropiate specification and returns any errors.
// Package files are obtained throug the given filesystem.
func ValidateFromFS(location string, fsys fs.FS) error {
	pkg, err := validator.NewPackageFromFS(location, fsys)
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

	if errs := spec.ValidatePackage(*pkg); errs != nil && len(errs) > 0 {
		return errs
	}

	return nil
}
