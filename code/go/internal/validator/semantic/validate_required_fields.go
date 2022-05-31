// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/pkg/errors"
)

// ValidateRequiredFields validates that required fields are present and have the expected
// types.
func ValidateRequiredFields(fsys fspath.FS) ve.ValidationErrors {
	requiredFields := map[string]string{
		"data_stream.type":      "constant_keyword",
		"data_stream.dataset":   "constant_keyword",
		"data_stream.namespace": "constant_keyword",
		"@timestamp":            "date",
	}

	return validateRequiredFields(fsys, requiredFields)
}

func validateRequiredFields(fsys fspath.FS, requiredFields map[string]string) ve.ValidationErrors {
	// map datastream -> field name -> found
	foundFields := make(map[string]map[string]struct{})

	dataStreams, err := listDataStreams(fsys)
	if err != nil {
		return ve.ValidationErrors{err}
	}

	checkField := func(fieldsFile string, f field) ve.ValidationErrors {
		expectedType, found := requiredFields[f.Name]
		if !found {
			return nil
		}

		datastream, err := dataStreamFromFieldsPath(fsys.Path(), fieldsFile)
		if err != nil {
			return ve.ValidationErrors{err}
		}

		if _, ok := foundFields[datastream]; !ok {
			foundFields[datastream] = make(map[string]struct{})
		}
		foundFields[datastream][f.Name] = struct{}{}

		// Don't check type for external fields.
		if f.External != "" {
			return nil
		}
		if f.Type != expectedType {
			return ve.ValidationErrors{errors.Errorf("expected type %q for required field %q, found %q in %q", expectedType, f.Name, f.Type, fieldsFile)}
		}

		return nil
	}
	errs := validateFields(fsys, checkField)

	for _, dataStream := range dataStreams {
		dataStreamFields := foundFields[dataStream]
		for requiredName, requiredType := range requiredFields {
			if _, found := dataStreamFields[requiredName]; !found {
				errs = append(errs, errors.Errorf("expected field %q with type %q not found in datastream %q", requiredName, requiredType, dataStream))
			}
		}
	}

	return errs
}
