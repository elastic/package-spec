// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"

	"github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	pve "github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// ValidateFieldGroups verifies if field groups don't have units and metric types defined.
func ValidateFieldGroups(fsys fspath.FS) pve.ValidationErrors {
	return validateFields(fsys, validateFieldUnit)
}

func validateFieldUnit(metadata fieldFileMetadata, f field) pve.ValidationErrors {
	if f.Type == "group" && f.Unit != "" {
		vError := errors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field "%s" can't have unit property'`, metadata.fullFilePath, f.Name),
			metadata.filePath,
			"",
			pve.Critical,
		)
		return pve.ValidationErrors{vError}
	}

	if f.Type == "group" && f.MetricType != "" {
		vError := errors.NewStructuredError(
			fmt.Errorf(`file "%s" is invalid: field "%s" can't have metric type property'`, metadata.fullFilePath, f.Name),
			metadata.filePath,
			"",
			pve.Critical,
		)
		return pve.ValidationErrors{vError}
	}

	return nil
}
