// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package processors

import (
	"regexp"

	"github.com/elastic/package-spec/v2/code/go/pkg/errors"
)

// Exclude is a processor to filter errors according to their messages.
type Exclude struct {
	pattern *regexp.Regexp
}

// NewExclude creates a new Exclude processor.
func NewExclude(pattern string) *Exclude {
	var patternRe *regexp.Regexp
	if pattern != "" {
		patternRe = regexp.MustCompile(pattern)
	}

	return &Exclude{
		pattern: patternRe,
	}
}

// Name returns the name of this Exclude processor.
func (p Exclude) Name() string {
	return "exclude"
}

// Process returns a new list of validation errors filtered.
func (p Exclude) Process(issues errors.ValidationErrors) (errors.ValidationErrors, error) {
	if p.pattern == nil {
		return issues, nil
	}

	errs, _ := issues.Filter(func(i error) bool {
		// if pathError, ok := i.(errors.ValidationPathError); ok
		return !p.pattern.MatchString(i.Error())
	})
	return errs, nil
}
