// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateFieldGroups verifies if field groups don't have units and metric types defined.
func ValidateFieldGroups(fsys fspath.FS) specerrors.ValidationErrors {
	return validateFields(fsys, validateFieldUnit)
}

func validateFieldUnit(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
	if f.Type == "group" && f.Unit != "" {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(`file "%s" is invalid: field "%s" can't have unit property'`, metadata.fullFilePath, f.Name),
		}
	}

	if f.Type == "group" && f.MetricType != "" {
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(`file "%s" is invalid: field "%s" can't have metric type property'`, metadata.fullFilePath, f.Name),
		}
	}

	return nil
}
