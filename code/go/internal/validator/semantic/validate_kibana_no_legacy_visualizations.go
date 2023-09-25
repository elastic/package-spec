// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"text/tabwriter"
	"strings"

	"github.com/elastic/kbncontent"
	ve "github.com/elastic/package-spec/v2/code/go/internal/errors"
	"github.com/elastic/package-spec/v2/code/go/internal/fspath"
	"github.com/elastic/package-spec/v2/code/go/internal/pkgpath"
)

// Reports legacy Kibana visualizations in a package.
func ValidateKibanaNoLegacyVisualizations(fsys fspath.FS) ve.ValidationErrors {
	var errs ve.ValidationErrors

	filePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, filePaths)
	if err != nil {
		errs = append(errs, fmt.Errorf("error finding Kibana dashboard files: %w", err))
		return errs
	}
	for _, file := range dashboardFiles {
		dashboardJSON, _ := file.Values("$")
		dashboardTitle, _ := kbncontent.GetDashboardTitle(dashboardJSON)
		panelsJSON, _ := file.Values("$.attributes.panelsJSON")
		visualizations, _ := kbncontent.DescribeByValueDashboardPanels(panelsJSON)

		var buf strings.Builder
		fmt.Fprint(&buf, "your package contains legacy visualizations:\n\n")
		tableWriter := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
		fmt.Fprintln(tableWriter, "\tDashboard title\tVisualization title\tVisualization editor\tVisualization type\t")
		fmt.Fprintln(tableWriter, "\t\t\t\t\t")
		hasLegacy := false
		for _, vis := range visualizations {
			if vis.IsLegacy {
				hasLegacy = true
				fmt.Fprintf(tableWriter, "\t\"%s\"\t\"%s\"\t%s\t%s\t\n", dashboardTitle, vis.Title, vis.Editor, vis.SemanticType)
			}
		}
		tableWriter.Flush()

		if hasLegacy {
			errs = append(errs, fmt.Errorf("%s", buf.String()))
		}
	}

	return errs
}
