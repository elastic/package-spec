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

	// Collect by-reference visualizations for reference later.
	// Note: this does not include Lens, Maps, or Discover. That's okay for this rule because none of those are legacy
	visFilePaths := path.Join("kibana", "visualization", "*.json")
	visFiles, _ := pkgpath.Files(fsys, visFilePaths)
	pkgByRefVisualizations := make(map[string]kbncontent.VisDesc)

	for _, file := range visFiles {
		if visJSON, err := file.Values("$"); err == nil {
			if desc, err := kbncontent.DescribeVisualizationSavedObject(visJSON.(map[string]interface{})); err == nil {
				pkgByRefVisualizations[strings.TrimSuffix(file.Name(), ".json")] = desc
			}
		}
	}

	dashboardFilePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, dashboardFilePaths)
	if err != nil {
		errs = append(errs, fmt.Errorf("error finding Kibana dashboard files: %w", err))
		return errs
	}

	for _, file := range dashboardFiles {
		dashboardJSON, _ := file.Values("$")
		panelsJSON, _ := file.Values("$.attributes.panelsJSON")
		visualizations, _ := kbncontent.DescribeByValueDashboardPanels(panelsJSON)

		var byRefVisualizations []kbncontent.VisDesc
		ids, _ := kbncontent.GetByReferencePanelIDs(dashboardJSON)
		for _, id := range ids {
			if vis, ok := pkgByRefVisualizations[id]; ok {
				byRefVisualizations = append(byRefVisualizations, vis)
			}
		}

		visualizations = append(visualizations, byRefVisualizations...)

		var legacyVisualizations []kbncontent.VisDesc
		for _, desc := range visualizations {
			if desc.IsLegacy {
				legacyVisualizations = append(legacyVisualizations, desc)
			}
		}

		if len(legacyVisualizations) > 0 {
			dashboardTitle, _ := kbncontent.GetDashboardTitle(dashboardJSON)
			var buf strings.Builder
			fmt.Fprintf(&buf, "Dashboard \"%s\" contains legacy visualizations:\n\n", dashboardTitle)

			tableWriter := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
			fmt.Fprintln(tableWriter, "\tTitle\tEditor\tType\t")
			fmt.Fprintln(tableWriter, "\t\t\t\t")

			for _, vis := range legacyVisualizations {
				fmt.Fprintf(tableWriter, "\t\"%s\"\t%s\t%s\t\n", vis.Title, vis.Editor, vis.SemanticType)
			}
			tableWriter.Flush()

			errs = append(errs, fmt.Errorf("%s", buf.String()))
		}
	}

	return errs
}
