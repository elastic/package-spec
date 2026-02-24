// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/package-spec/v3/code/go/internal/fspath"
	"github.com/elastic/package-spec/v3/code/go/internal/pkgpath"
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
			valid:     false,
		},
		{
			dashboard: "testdata/dashboards/tomcat-with-filter.json",
			valid:     true,
		},

		// Embedded JSON objects in dashboards are encoded in built packages.
		{
			dashboard: "testdata/dashboards/apache-no-filter-encoded.json",
			valid:     false,
		},
		{
			dashboard: "testdata/dashboards/nats-with-query-encoded.json",
			valid:     false,
		},
		{
			dashboard: "testdata/dashboards/tomcat-with-filter-encoded.json",
			valid:     true,
		},

		// Dashboard has the filter in all panels.
		{
			dashboard: "testdata/dashboards/mysql-filter-in-panels.json",
			valid:     true,
		},
		{
			dashboard: "testdata/dashboards/mysql-filter-in-panels-encoded.json",
			valid:     true,
		},

		// ES|QL panels don't use traditional filters; they should pass.
		{
			dashboard: "testdata/dashboards/esql-no-filter.json",
			valid:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.dashboard, func(t *testing.T) {
			files, err := pkgpath.Files(fspath.DirFS("."), c.dashboard)
			require.NoError(t, err)
			require.Len(t, files, 1, "looking for %s", c.dashboard)

			err = checkDashboardHasFilter(files[0])
			if err != nil {
				t.Log(err)
			}
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
