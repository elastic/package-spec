// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDashboardHasFilter(t *testing.T) {
	cases := []struct {
		dashboard string
		valid     bool
	}{
		{
			dashboard: "testdata/dashboards/apache-no-filter.json",
			valid:     false,
		},
		{
			dashboard: "testdata/dashboards/nats-with-query.json",
			valid:     true,
		},
		{
			dashboard: "testdata/dashboards/tomcat-with-filter.json",
			valid:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.dashboard, func(t *testing.T) {
			files, err := pkgpath.Files(fspath.DirFS("."), c.dashboard)
			require.NoError(t, err)
			require.Len(t, files, 1)

			err = checkDashboardHasFilter(files[0])
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
