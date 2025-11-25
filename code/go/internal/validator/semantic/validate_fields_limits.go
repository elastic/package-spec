// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"log"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateFieldsLimits verifies limits on fields.
func ValidateFieldsLimits(limit int) func(fspath.FS) specerrors.ValidationErrors {
	return func(fsys fspath.FS) specerrors.ValidationErrors {
		return validateFieldsLimits(fsys, limit)
	}
}

func validateFieldsLimits(fsys fspath.FS, limit int) specerrors.ValidationErrors {
	counts := make(map[string]int)
	// Created a new map to avoid collisions with data stream names
	transformCounts := make(map[string]int)
	countField := func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		if metadata.transform != "" {
			count := transformCounts[metadata.transform]
			transformCounts[metadata.transform] = count + 1
			return nil
		}

		count := counts[metadata.dataStream]
		counts[metadata.dataStream] = count + 1
		return nil
	}

	err := validateFields(fsys, countField)
	if err != nil {
		return err
	}
	var errs specerrors.ValidationErrors
	for id, count := range counts {
		log.Println("Data stream:", id, "Count:", count)
		if count > limit {
			if id != "" {
				errs = append(errs, specerrors.NewStructuredErrorf("data stream %s has more than %d fields (%d)", id, limit, count))
			} else {
				errs = append(errs, specerrors.NewStructuredErrorf("input package has more than %d fields (%d)", limit, count))
			}
		}
	}
	for id, count := range transformCounts {
		if count > limit {
			errs = append(errs, specerrors.NewStructuredErrorf("transform %s has more than %d fields (%d)", id, limit, count))
		}
	}
	return errs
}
