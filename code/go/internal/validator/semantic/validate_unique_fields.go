// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
)

// ValidateUniqueFields verifies that any field is defined only once on each data stream.
func ValidateUniqueFields(fsys fspath.FS) ve.ValidationErrors {
	// data_stream -> field -> files
	fields := make(map[string]map[string][]string)

	countField := func(metadata fieldFileMetadata, f field) ve.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		id := metadata.ID()

		dsMap, found := fields[id]
		if !found {
			dsMap = make(map[string][]string)
			fields[id] = dsMap
		}
		dsMap[f.Name] = append(dsMap[f.Name], metadata.filePath)
		return nil
	}

	err := validateFields(fsys, countField)
	if err != nil {
		return err
	}

	var errs ve.ValidationErrors
	for id, defs := range fields {
		for field, files := range defs {
			if len(files) > 1 {
				sort.Strings(files)
				errs = append(errs,
					errors.Errorf("field %q is defined multiple times for data stream %q, found in: %s",
						field, id, strings.Join(files, ", ")))
			}
		}

	}
	return errs
}
