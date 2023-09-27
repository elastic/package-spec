// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package semantic

import (
	"fmt"
	"path"
	"text/tabwriter"
	"strings"
	"errors"

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

	for _, file := range visFiles {
		visJSON, err := file.Values("$")

		if err != nil {
			errs = append(errs, fmt.Errorf("error getting JSON: %w", err))
			return errs
		}


		desc, err := kbncontent.DescribeVisualizationSavedObject(visJSON.(map[string]interface{}))
		if err != nil {
			errs = append(errs, fmt.Errorf("error describing visualization saved object: %w", err))
		}

		if desc.IsLegacy() {
			var editor string
			if result, err := desc.Editor(); err == nil {
				editor = result
			}
			errs = append(errs, fmt.Errorf("file \"%s\" is invalid: found legacy visualization \"%s\" (%s, %s)", fsys.Path(file.Path()), desc.Title(), desc.SemanticType(), editor))
		}
	}

	dashboardFilePaths := path.Join("kibana", "dashboard", "*.json")
	dashboardFiles, err := pkgpath.Files(fsys, dashboardFilePaths)
	if err != nil {
		errs = append(errs, fmt.Errorf("error finding Kibana dashboard files: %w", err))
		return errs
	}

	for _, file := range dashboardFiles {
		dashboardJSON, err := file.Values("$")
		if err != nil {
			errs = append(errs, fmt.Errorf("error getting dashboard JSON: %w", err))
			return errs
		}

		visualizations, err := kbncontent.DescribeByValueDashboardPanels(dashboardJSON)
		if err != nil {
			errs = append(errs, fmt.Errorf("error describing dashboard panels for %s: %w", fsys.Path(file.Path()), err))
			return errs
		}

		var legacyVisualizations []kbncontent.VisualizationDescriptor
		for _, desc := range visualizations {
			if desc.IsLegacy() {
				legacyVisualizations = append(legacyVisualizations, desc)
			}
		}

		if len(legacyVisualizations) > 0 {
			dashboardTitle, err := kbncontent.GetDashboardTitle(dashboardJSON)
			if err != nil {
				errs = append(errs, fmt.Errorf("error fetching dashboard title: %w", err))
				return errs
			}

			var buf strings.Builder
			fmt.Fprintf(&buf, "file \"%s\" is invalid: \"%s\" contains legacy visualizations:\n\n", fsys.Path(file.Path()), dashboardTitle)

			tableWriter := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)
			fmt.Fprintln(tableWriter, "\tTitle\tEditor\tType\t")
			fmt.Fprintln(tableWriter, "\t\t\t\t")

			for _, vis := range legacyVisualizations {
				var editor string
				if result, err := vis.Editor(); err == nil {
					editor = result
				}
				fmt.Fprintf(tableWriter, "\t\"%s\"\t%s\t%s\t\n", vis.Title(), editor, vis.SemanticType())
			}
			tableWriter.Flush()

			errs = append(errs, errors.New(buf.String()))
		}
	}

	return errs
}
