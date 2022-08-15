// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package validator

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	ve "github.com/elastic/package-spec/code/go/internal/errors"
	"github.com/elastic/package-spec/code/go/internal/fspath"
	"github.com/elastic/package-spec/code/go/internal/spectypes"
	"github.com/elastic/package-spec/code/go/internal/validator/semantic"
)

type validationRulesBuilder func(rootSpec spectypes.ItemSpec) validationRules

type validationRules []func(pkg fspath.FS) ve.ValidationErrors

func (vr validationRules) validate(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors
	for _, validationRule := range vr {
		err := validationRule(fsys)
		errs.Append(err)
	}

	return errs
}

func newRulesBuilder(version semver.Version) (validationRulesBuilder, error) {
	switch version.Major() {
	case 0, 1:
		return func(rootSpec spectypes.ItemSpec) validationRules {
			return validationRules{
				semantic.ValidateKibanaObjectIDs,
				semantic.ValidateVersionIntegrity,
				semantic.ValidateChangelogLinks,
				semantic.ValidatePrerelease,
				semantic.ValidateFieldGroups,
				semantic.ValidateFieldsLimits(rootSpec.MaxFieldsPerDataStream()),
				semantic.ValidateDimensionFields,
				semantic.ValidateRequiredFields,
			}
		}, nil
	case 2:
		return func(rootSpec spectypes.ItemSpec) validationRules {
			return validationRules{
				semantic.ValidateKibanaObjectIDs,
				semantic.ValidateVersionIntegrity,
				semantic.ValidateChangelogLinks,
				semantic.ValidatePrerelease,
				semantic.ValidateFieldGroups,
				semantic.ValidateFieldsLimits(rootSpec.MaxFieldsPerDataStream()),
				// Temporarily disabled: https://github.com/elastic/package-spec/issues/331
				//semantic.ValidateUniqueFields,
				semantic.ValidateUniqueFields,
				semantic.ValidateDimensionFields,
				semantic.ValidateRequiredFields,
			}
		}, nil
	}

	return nil, fmt.Errorf("no rules defined for version %v", version)
}
