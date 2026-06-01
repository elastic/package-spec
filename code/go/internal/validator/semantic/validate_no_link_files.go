// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"
	"path/filepath"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateNoLinkFiles errors for any .link file found in the package.
// .link files are a source-only artifact used during development to share
// files across packages. They must be resolved and inlined by the build
// step before a package is distributed. A built package must not contain
// any .link files when validated with ModeBuild.
func ValidateNoLinkFiles(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	walkErr := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".link" {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q: .link files are not allowed in built packages",
				fsys.Path(p),
			))
		}
		return nil
	})
	if walkErr != nil {
		errs = append(errs, specerrors.NewStructuredError(walkErr, specerrors.UnassignedCode))
	}
	return errs
}
