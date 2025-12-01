// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateRequiredFields validates that required fields are present and have the expected
// types except for fields defined in transforms.
func ValidateRequiredFields(fsys fspath.FS) specerrors.ValidationErrors {
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
	fullPath     string
}

func (e unexpectedTypeRequiredField) Error() string {
	return fmt.Sprintf("expected type %q for required field %q, found %q in %q", e.expectedType, e.field, e.foundType, e.fullPath)
}

type notFoundRequiredField struct {
	field        string
	expectedType string
	dataStream   string
	transform    string
}

func (e notFoundRequiredField) Error() string {
	message := fmt.Sprintf("expected field %q", e.field)
	if e.expectedType != "" {
		message = fmt.Sprintf("%s with type %q", message, e.expectedType)
	}
	message = fmt.Sprintf("%s not found", message)
	if e.dataStream != "" {
		message = fmt.Sprintf("%s in datastream %q", message, e.dataStream)
	}
	if e.transform != "" {
		message = fmt.Sprintf("%s in transform %q", message, e.transform)
	}
	return message
}

func validateRequiredFields(fsys fspath.FS, requiredFields map[string]string) specerrors.ValidationErrors {
	// map datastream/input package -> field name -> found
	// if data stream is an empty string, it means it is an input package
	foundFields := make(map[string]map[string]struct{})

	checkField := func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if metadata.transform != "" {
			// Skip required fields check for fields found in transforms
			// as they are not mandatory there.
			return nil
		}
		// It is created a key with an empty string if it is an input package,
		// since input packages don't have data streams.
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
			return specerrors.ValidationErrors{
				specerrors.NewStructuredError(
					unexpectedTypeRequiredField{
						field:        f.Name,
						foundType:    f.Type,
						fullPath:     metadata.fullFilePath,
						expectedType: expectedType,
					},
					specerrors.UnassignedCode,
				),
			}
		}

		return nil
	}
	errs := validateFields(fsys, checkField)

	// Validate that required fields exist in integration and input packages.
	// Using the data streams found here, since all data streams must have a `fields` folder
	// If a data stream is an empty string, it means it is an input package
	// Fields folder is not mandatory for input packages, so we need to consider the case
	// https://github.com/elastic/package-spec/pull/994
	for dataStream, dataStreamFields := range foundFields {
		for requiredName, requiredType := range requiredFields {
			if _, found := dataStreamFields[requiredName]; !found {
				errs = append(errs,
					specerrors.NewStructuredError(
						notFoundRequiredField{
							field:        requiredName,
							expectedType: requiredType,
							dataStream:   dataStream,
						},
						specerrors.UnassignedCode,
					),
				)
			}
		}
	}

	return errs
}
