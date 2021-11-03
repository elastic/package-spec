// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDimensionFields(t *testing.T) {
	cases := []struct {
		title string
		field field
		valid bool
	}{
		{
			title: "usual keyword dimension",
			field: field{
				Name:      "host.id",
				Type:      "keyword",
				Dimension: true,
			},
			valid: true,
		},
		{
			title: "not a dimension",
			field: field{
				Name: "host.id",
				Type: "histogram",
			},
			valid: true,
		},
		{
			title: "ip dimension",
			field: field{
				Name:      "source.ip",
				Type:      "ip",
				Dimension: true,
			},
			valid: true,
		},
		{
			title: "numeric dimension",
			field: field{
				Name:      "http.body.size",
				Type:      "long",
				Dimension: true,
			},
			valid: true,
		},
		{
			title: "histogram dimension is not supported",
			field: field{
				Name:      "http.response.time",
				Type:      "histogram",
				Dimension: true,
			},
			valid: false,
		},
		{
			title: "nested field as dimension is not supported",
			field: field{
				Name:      "process.child",
				Type:      "nested",
				Dimension: true,
			},
			valid: false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			errs := validateDimensionField("fields.yml", c.field)
			if c.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}
