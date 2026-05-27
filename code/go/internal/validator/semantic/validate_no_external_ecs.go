// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/pkg/specerrors"
)

// ValidateNoExternalEcs errors for any field referencing external: ecs.
// The build process materializes ECS field references — once built, fields
// should carry full definitions, not external pointers. A built package must
// not contain any fields with external: ecs when validated with ModeBuild.
func ValidateNoExternalEcs(fsys fspath.FS) specerrors.ValidationErrors {
	validateFunc := func(metadata fieldFileMetadata, f field) specerrors.ValidationErrors {
		if f.External != "ecs" {
			return nil
		}
		return specerrors.ValidationErrors{
			specerrors.NewStructuredErrorf(
				"file \"%s\" is invalid: field %s has external: ecs reference; ECS fields must be materialized before packaging",
				metadata.fullFilePath, f.Name,
			),
		}
	}
	return validateFields(fsys, validateFunc)
}
