// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDateFields(t *testing.T) {
	cases := []struct {
		title string
		field field
		valid bool
	}{
		{
			title: "non-date the define date format",
			field: field{
				Name:       "my_keyword",
				Type:       "keyword",
				DateFormat: "yyyy-MM-dd",
			},
			valid: false,
		},
		{
			title: "date the define date format",
			field: field{
				Name:       "my_date",
				Type:       "date",
				DateFormat: "yyyy-MM-dd",
			},
			valid: true,
		},
		{
			title: "date without date format",
			field: field{
				Name: "my_date",
				Type: "date",
			},
			valid: true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errs := validateDateField("fields.yml", c.field)
			if c.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}
