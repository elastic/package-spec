// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	ve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateFieldsLimits verifies limits on fields.
func ValidateFieldsLimits(limit int) func(fspath.FS) ve.ValidationErrors {
	return func(fsys fspath.FS) ve.ValidationErrors {
		return validateFieldsLimits(fsys, limit)
	}
}

func validateFieldsLimits(fsys fspath.FS, limit int) ve.ValidationErrors {
	counts := make(map[string]int)
	countField := func(metadata fieldFileMetadata, f field) ve.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		count, _ := counts[metadata.dataStream]
		counts[metadata.dataStream] = count + 1
		return nil
	}

	err := validateFields(fsys, countField)
	if err != nil {
		return err
	}

	var errs ve.ValidationErrors
	for id, count := range counts {
		if count > limit {
			errs = append(errs, fmt.Errorf("data stream %s has more than %d fields (%d)", id, limit, count))
		}
	}
	return errs
}
