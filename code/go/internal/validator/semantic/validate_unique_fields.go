// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"sort"
	"strings"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/pkg/errors"
)

// ValidateUniqueFields verifies that any field is defined only once on each data stream.
func ValidateUniqueFields(fsys fspath.FS) ve.ValidationErrors {
	// data_stream -> field -> files
	fields := make(map[string]map[string][]string)

	countField := func(fieldsFile string, f field) ve.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		dataStream, err := dataStreamFromFieldsPath(fsys.Path(), fieldsFile)
		if err != nil {
			return ve.ValidationErrors{err}
		}

		dsMap, found := fields[dataStream]
		if !found {
			dsMap = make(map[string][]string)
			fields[dataStream] = dsMap
		}
		dsMap[f.Name] = append(dsMap[f.Name], fieldsFile)
		return nil
	}

	err := validateFields(fsys, countField)
	if err != nil {
		return err
	}

	var errs ve.ValidationErrors
	for dataStream, dataStreamFields := range fields {
		for field, files := range dataStreamFields {
			if len(files) > 1 {
				sort.Strings(files)
				errs = append(errs,
					errors.Errorf("field %q is defined multiple times for data stream %q, found in: %s",
						field, dataStream, strings.Join(files, ", ")))
			}
		}

	}
	return errs
}
