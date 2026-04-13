// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
)

func TestValidateDatastreamPackageCategories(t *testing.T) {
	cases := []struct {
		title        string
		setup        func(t *testing.T, dir string)
		expectedErrs []string
	}{
		{
			title: "package includes datastream parent category",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - security
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
type: logs
`)
			},
		},
		{
			title: "package missing datastream parent category",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
type: logs
`)
			},
			expectedErrs: []string{
				`package manifest categories [observability] are missing parent categories [security] from data stream "mylogs"`,
			},
		},
		{
			title: "datastream subcategory requires its parent in package",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - credential_management
type: logs
`)
			},
			expectedErrs: []string{
				`package manifest categories [observability] are missing parent categories [security] from data stream "mylogs"`,
			},
		},
		{
			title: "datastream subcategory parent already in package",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - security
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - credential_management
type: logs
`)
			},
		},
		{
			title: "data stream manifest has no categories field, skipped",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
type: logs
`)
			},
		},
		{
			title: "package has no categories but datastream does",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
type: logs
`)
			},
			expectedErrs: []string{
				`package manifest categories [] are missing parent categories [security] from data stream "mylogs"`,
			},
		},
		{
			title: "package has subset, missing one datastream parent category",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - security
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
  - observability
type: logs
`)
			},
			expectedErrs: []string{`are missing parent categories [observability] from data stream "mylogs"`},
		},
		{
			title: "non-integration packages are skipped",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: input
categories:
  - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
type: logs
`)
			},
		},
		{
			title: "multiple datastreams, one triggers error",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - security
`)
				writeDataStreamManifest(t, dir, "auditlogs", `
title: Audit Logs
categories:
  - security
type: logs
`)
				writeDataStreamManifest(t, dir, "netlogs", `
title: Net Logs
categories:
  - network
type: logs
`)
			},
			expectedErrs: []string{`are missing parent categories [network] from data stream "netlogs"`},
		},
		{
			title: "datastream does not need all package categories",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, `
type: integration
categories:
  - security
  - observability
`)
				writeDataStreamManifest(t, dir, "mylogs", `
title: My Logs
categories:
  - security
type: logs
`)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)

			errs := ValidateDatastreamPackageCategories(fspath.DirFS(dir))

			if len(tc.expectedErrs) == 0 {
				assert.Empty(t, errs)
			} else {
				require.Len(t, errs, 1)
				for _, expected := range tc.expectedErrs {
					assert.Contains(t, errs[0].Error(), expected)
				}
			}
		})
	}
}
