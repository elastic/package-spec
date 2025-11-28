// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

type uniqueField struct {
	name       string
	dataStream string
	transform  string
}

// ValidateUniqueFields verifies that any field is defined only once on each data stream.
func ValidateUniqueFields(fsys fspath.FS) specerrors.ValidationErrors {
	// data_stream -> field -> files
	// if data stream is empty string, it means it is an input package
	fields := make(map[string]map[uniqueField][]string)
	// transform -> field -> files
	// Created a new map to avoid collisions with data stream names
	transformFields := make(map[string]map[uniqueField][]string)

	countField := func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if len(f.Fields) > 0 {
			// Don't count groups
			return nil
		}

		if metadata.transform != "" {
			transformMap, found := transformFields[metadata.transform]
			if !found {
				transformMap = make(map[uniqueField][]string)
				transformFields[metadata.transform] = transformMap
			}
			field := uniqueField{
				name:      f.Name,
				transform: metadata.transform,
			}
			transformMap[field] = append(transformMap[field], metadata.fullFilePath)
			return nil
		}

		dsMap, found := fields[metadata.dataStream]
		if !found {
			dsMap = make(map[uniqueField][]string)
			fields[metadata.dataStream] = dsMap
		}
		field := uniqueField{
			name:       f.Name,
			dataStream: metadata.dataStream,
		}
		dsMap[field] = append(dsMap[field], metadata.fullFilePath)
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
				message := fmt.Sprintf("field %q is defined multiple times", field.name)
				if id != "" && field.dataStream != "" {
					message += fmt.Sprintf(" for data stream %q", id)
				}
				errs = append(errs,
					specerrors.NewStructuredErrorf("%s, found in: %s", message, strings.Join(files, ", ")),
				)
			}
		}
	}

	for id, defs := range transformFields {
		for field, files := range defs {
			if len(files) > 1 {
				sort.Strings(files)
				message := fmt.Sprintf("field %q is defined multiple times", field.name)
				if id != "" && field.transform != "" {
					message += fmt.Sprintf(" for transform %q", id)
				}
				errs = append(errs,
					specerrors.NewStructuredErrorf("%s, found in: %s", message, strings.Join(files, ", ")),
				)
			}
		}
	}

	return errs
}
