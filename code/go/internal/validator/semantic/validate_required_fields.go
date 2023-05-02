// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
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

type unexpectedTypeRequiredField struct {
	field        string
	expectedType string
	foundType    string
	dataStream   string
	fullPath     string
}

func (e unexpectedTypeRequiredField) Error() string {
	return fmt.Sprintf("expected type %q for required field %q, found %q in %q", e.expectedType, e.field, e.foundType, e.fullPath)
}

type notFoundRequiredField struct {
	field        string
	expectedType string
	dataStream   string
}

func (e notFoundRequiredField) Error() string {
	message := fmt.Sprintf("expected field %q with type %q not found", e.field, e.expectedType)
	if e.dataStream != "" {
		message = fmt.Sprintf("%s in datastream %q", message, e.dataStream)
	}
	return message
}

func validateRequiredFields(fsys fspath.FS, requiredFields map[string]string) ve.ValidationErrors {
	// map datastream/package -> field name -> found
	foundFields := make(map[string]map[string]struct{})

	checkField := func(metadata fieldFileMetadata, f field) ve.ValidationErrors {
		if _, ok := foundFields[metadata.dataStream]; !ok {
			foundFields[metadata.dataStream] = make(map[string]struct{})
		}
		foundFields[metadata.dataStream][f.Name] = struct{}{}

		expectedType, found := requiredFields[f.Name]
		if !found {
			return nil
		}

		// Check if type is the expected one, but only for fields what are
		// not declared as external. External fields won't have a type in
		// the definition.
		// More info in https://github.com/elastic/elastic-package/issues/749
		if f.External == "" && f.Type != expectedType {
			return ve.ValidationErrors{
				unexpectedTypeRequiredField{
					field:        f.Name,
					foundType:    f.Type,
					dataStream:   metadata.dataStream,
					fullPath:     metadata.fullFilePath,
					expectedType: expectedType,
				},
			}
		}

		return nil
	}
	errs := validateFields(fsys, checkField)

	// Using the data streams found here, since there could not be a data stream
	// without the `fields` folder or an input package without that folder
	for dataStream, dataStreamFields := range foundFields {
		for requiredName, requiredType := range requiredFields {
			if _, found := dataStreamFields[requiredName]; !found {
				errs = append(errs,
					notFoundRequiredField{
						field:        requiredName,
						expectedType: requiredType,
						dataStream:   dataStream,
					},
				)
			}
		}
	}

	return errs
}
