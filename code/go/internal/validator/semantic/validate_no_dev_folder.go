// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"io/fs"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateNoDevFolder errors for any _dev/ directory found in the package.
// _dev/ is a source-only artifact used during development (tests, deploy
// configs, build manifests). It must not appear in a built package that is
// validated with ModeBuild.
func ValidateNoDevFolder(fsys fspath.FS) specerrors.ValidationErrors {
	var errs specerrors.ValidationErrors
	walkErr := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "_dev" {
			errs = append(errs, specerrors.NewStructuredErrorf(
				"file %q: _dev directory is not allowed in built packages",
				fsys.Path(p),
			))
			// Skip the subtree to avoid generating child errors for each
			// file inside the _dev directory.
			return fs.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		errs = append(errs, specerrors.NewStructuredError(walkErr, specerrors.UnassignedCode))
	}
	return errs
}
