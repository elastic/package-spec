// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"sort"
	"strings"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/pkg/specerrors"
)

// ValidateUniqueFields verifies that any field is defined only once on each data stream.
func ValidateUniqueFields(fsys fspath.FS) specerrors.ValidationErrors {
	// data_stream -> field -> files
	fields := make(map[string]map[string][]string)

	countField := func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		dsMap, found := fields[metadata.dataStream]
		if !found {
			dsMap = make(map[string][]string)
			fields[metadata.dataStream] = dsMap
		}
		dsMap[f.Name] = append(dsMap[f.Name], metadata.fullFilePath)
		return nil
	}

	err := validateFields(fsys, countField)
	if err != nil {
		return err
	}

	var errs specerrors.ValidationErrors
	for id, defs := range fields {
		for field, files := range defs {
			if len(files) > 1 {
				sort.Strings(files)
				errs = append(errs,
					specerrors.NewStructuredErrorf(
						"field %q is defined multiple times for data stream %q, found in: %s",
						field, id, strings.Join(files, ", ")),
				)
			}
		}
	}
	return errs
}
