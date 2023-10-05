// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"log"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/validator/common"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// WarnOn returns a validation function that wraps another one. Errors returned by the
// wrapped validation that have a filtering code are printed as warnings. Other errors
// are directly returned.
func WarnOn(validation func(fsys fspath.FS) specerrors.ValidationErrors) func(fspath.FS) specerrors.ValidationErrors {
	return func(fsys fspath.FS) specerrors.ValidationErrors {
		errs := validation(fsys)
		if common.IsDefinedWarningsAsErrors() {
			return errs
		}

		k := 0
		for i := range errs {
			if err := errs[i]; err.Code() != specerrors.UnassignedCode {
				log.Printf("Warning: %s", err.Error())
				continue
			}

			errs[k] = errs[i]
			k++
		}

		return errs[:k]
	}
}
