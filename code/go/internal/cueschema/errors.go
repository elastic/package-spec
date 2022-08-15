// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cueschema

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
)

var (
	reEmptyDisjunctionErr  = regexp.MustCompile(`^(.*): (\d+) errors in empty disjunction`)
	reConflictingValuesErr = regexp.MustCompile(`^(.*): conflicting values (.*) and (.*)`)
)

// validationErrors transforms cue errors into more human-friendly errors.
func validationErrors(filename string, err error) []error {
	var result []error
	for i, errs := 0, cueerrors.Errors(err); i < len(errs); i++ {
		e := errs[i]
		if m := reEmptyDisjunctionErr.FindStringSubmatch(e.Error()); len(m) > 0 {
			n, _ := strconv.Atoi(string(m[2]))
			builder := validationErrorBuilder{
				Filename:  filename,
				Field:     m[1],
				Conflicts: errs[i+1 : i+n],
			}
			err := builder.Build()
			result = append(result, err)
			i += n
			continue
		}

		pos := positionForError(e)
		err := fmt.Errorf("%s:%d:%d: %s", filename, pos.Line(), pos.Column(), e)
		result = append(result, err)
	}
	return result
}

type validationErrorBuilder struct {
	Filename  string
	Field     string
	Conflicts []cueerrors.Error
}

func (b *validationErrorBuilder) Build() error {
	pos := positionForError(b.Conflicts[0])
	var expected []string
	var found string
	for _, conflict := range b.Conflicts {
		m := reConflictingValuesErr.FindStringSubmatch(conflict.Error())
		expected = append(expected, string(m[2]))
		if found == "" {
			found = string(m[3])
		}
	}
	return fmt.Errorf("%s:%d:%d: %s: found %s, expected one of: %s",
		b.Filename, pos.Line(), pos.Column(), b.Field,
		found, strings.Join(expected, ", "),
	)
}

func positionForError(err cueerrors.Error) token.Pos {
	for _, pos := range err.InputPositions() {
		// YAML filename is empty.
		if pos.Filename() == "" {
			return pos
		}
	}
	return err.Position()
}
